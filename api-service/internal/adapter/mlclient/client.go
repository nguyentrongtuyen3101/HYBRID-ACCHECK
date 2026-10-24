// Package mlclient hiện thực app.Analyzer bằng cách gọi ML service (Python) qua HTTP.
package mlclient

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"reqsentinel/api/internal/domain"
)

type Client struct {
	baseURL string
	token   string
	http    *http.Client
}

func New(baseURL, token string, timeout time.Duration) *Client {
	return &Client{baseURL: baseURL, token: token, http: &http.Client{Timeout: timeout}}
}

// ---------- Hợp đồng JSON với accheck_ml (PipelineInput / PipelineOutput) ----------

type pmRowJSON struct {
	Role      string  `json:"role"`
	Action    string  `json:"action"`
	Resource  string  `json:"resource"`
	Effect    string  `json:"effect"`
	Condition *string `json:"condition"`
	Scope     *string `json:"scope"`
}

type inputJSON struct {
	UserStory          string      `json:"user_story"`
	AcceptanceCriteria []string    `json:"acceptance_criteria"`
	PermissionMatrix   []pmRowJSON `json:"permission_matrix"`
}

type entityJSON struct {
	Role       *string `json:"role"`
	Action     *string `json:"action"`
	Resource   *string `json:"resource"`
	Effect     *string `json:"effect"`
	Condition  *string `json:"condition"`
	Scope      *string `json:"scope"`
	Source     string  `json:"source"`
	Confidence float64 `json:"confidence"`
}

type alignmentJSON struct {
	Extracted       entityJSON `json:"extracted"`
	MatchedPMRow    *pmRowJSON `json:"matched_pm_row"`
	SimilarityScore float64    `json:"similarity_score"`
	IsMatched       bool       `json:"is_matched"`
}

type conflictJSON struct {
	ConflictType    string       `json:"conflict_type"`
	Confidence      float64      `json:"confidence"`
	Explanation     string       `json:"explanation"`
	EvidenceUSAC    *string      `json:"evidence_us_ac"`
	EvidencePM      *string      `json:"evidence_pm"`
	RelatedEntities []entityJSON `json:"related_entities"`
}

type outputJSON struct {
	ExtractedEntities []entityJSON    `json:"extracted_entities"`
	AlignmentResults  []alignmentJSON `json:"alignment_results"`
	Conflicts         []conflictJSON  `json:"conflicts"`
	IsConsistent      bool            `json:"is_consistent"`
	Summary           string          `json:"summary"`
	ModelVersion      string          `json:"model_version"`
}

func (c *Client) Analyze(ctx context.Context, in domain.AnalysisInput) (domain.AnalysisResult, error) {
	body, err := json.Marshal(toInputJSON(in))
	if err != nil {
		return domain.AnalysisResult{}, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/v1/analyze", bytes.NewReader(body))
	if err != nil {
		return domain.AnalysisResult{}, err
	}
	req.Header.Set("Content-Type", "application/json")
	if c.token != "" {
		req.Header.Set("X-Internal-Token", c.token)
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return domain.AnalysisResult{}, unavailable(err)
	}
	defer resp.Body.Close()
	raw, err := io.ReadAll(io.LimitReader(resp.Body, 10<<20))
	if err != nil {
		return domain.AnalysisResult{}, unavailable(err)
	}
	switch {
	case resp.StatusCode == http.StatusUnauthorized:
		return domain.AnalysisResult{}, unavailable(fmt.Errorf("internal token rejected (check ML_SERVICE_TOKEN)"))
	case resp.StatusCode >= 500:
		return domain.AnalysisResult{}, unavailable(fmt.Errorf("status %d", resp.StatusCode))
	case resp.StatusCode >= 400:
		return domain.AnalysisResult{}, domain.Validation("analysis_rejected", "Analysis service rejected input: "+string(raw))
	}
	var out outputJSON
	if err := json.Unmarshal(raw, &out); err != nil {
		return domain.AnalysisResult{}, unavailable(fmt.Errorf("invalid response: %w", err))
	}
	return toDomain(out), nil
}

func unavailable(err error) error {
	return domain.Unavailable("analysis_unavailable",
		"Analysis service is unavailable, please try again later ("+err.Error()+")")
}

// Health trả về JSON /health của ML service.
func (c *Client) Health(ctx context.Context) (map[string]any, error) {
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+"/health", nil)
	if err != nil {
		return nil, err
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("status %d", resp.StatusCode)
	}
	var out map[string]any
	err = json.NewDecoder(resp.Body).Decode(&out)
	return out, err
}

// ---------- mapping ----------

func toInputJSON(in domain.AnalysisInput) inputJSON {
	out := inputJSON{UserStory: in.UserStory, AcceptanceCriteria: in.AcceptanceCriteria, PermissionMatrix: []pmRowJSON{}}
	if out.AcceptanceCriteria == nil {
		out.AcceptanceCriteria = []string{}
	}
	for _, r := range in.PermissionMatrix {
		out.PermissionMatrix = append(out.PermissionMatrix, pmRowJSON{r.Role, r.Action, r.Resource, string(r.Effect), r.Condition, r.Scope})
	}
	return out
}

func toEntity(e entityJSON) domain.ExtractedEntity {
	return domain.ExtractedEntity{Role: e.Role, Action: e.Action, Resource: e.Resource, Effect: e.Effect,
		Condition: e.Condition, Scope: e.Scope, Source: e.Source, Confidence: e.Confidence}
}

func toDomain(o outputJSON) domain.AnalysisResult {
	res := domain.AnalysisResult{IsConsistent: o.IsConsistent, Summary: o.Summary, ModelVersion: o.ModelVersion,
		Entities: []domain.ExtractedEntity{}, Alignments: []domain.Alignment{}, Conflicts: []domain.DetectedConflict{}}
	for _, e := range o.ExtractedEntities {
		res.Entities = append(res.Entities, toEntity(e))
	}
	for _, a := range o.AlignmentResults {
		al := domain.Alignment{Extracted: toEntity(a.Extracted), Score: a.SimilarityScore, IsMatched: a.IsMatched}
		if m := a.MatchedPMRow; m != nil {
			al.Matched = &domain.PMRow{Role: m.Role, Action: m.Action, Resource: m.Resource,
				Effect: domain.Effect(m.Effect), Condition: m.Condition, Scope: m.Scope}
		}
		res.Alignments = append(res.Alignments, al)
	}
	for _, c := range o.Conflicts {
		dc := domain.DetectedConflict{Type: domain.ConflictType(c.ConflictType), Confidence: c.Confidence,
			Explanation: c.Explanation, EvidenceUSAC: c.EvidenceUSAC, EvidencePM: c.EvidencePM}
		for _, e := range c.RelatedEntities {
			dc.Related = append(dc.Related, toEntity(e))
		}
		res.Conflicts = append(res.Conflicts, dc)
	}
	return res
}
