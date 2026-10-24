// Package domain chứa entity, value object và luật nghiệp vụ thuần.
// Không import DB, HTTP hay bất kỳ framework nào.
package domain

import "errors"

type ErrorKind int

const (
	KindInternal ErrorKind = iota
	KindValidation
	KindNotFound
	KindForbidden
	KindConflict
	KindUnauthorized
	KindUnavailable
	KindRateLimited
)

// Error là lỗi nghiệp vụ có phân loại.
//   - Kind: adapter (HTTP) map sang status code
//   - Code: mã ổn định để client tra bản dịch (i18n), vd "project_not_found"
//   - Msg: mô tả tiếng Anh cho log / client không dịch
//   - Params: giá trị chèn vào bản dịch, vd {"min": 8}
type Error struct {
	Kind   ErrorKind
	Code   string
	Msg    string
	Params map[string]any
}

func (e *Error) Error() string { return e.Msg }

// With trả về bản sao kèm tham số cho bản dịch.
func (e *Error) With(params map[string]any) *Error {
	cp := *e
	cp.Params = params
	return &cp
}

func newErr(kind ErrorKind, code, msg string) *Error { return &Error{Kind: kind, Code: code, Msg: msg} }

func Validation(code, msg string) *Error   { return newErr(KindValidation, code, msg) }
func NotFound(code, msg string) *Error     { return newErr(KindNotFound, code, msg) }
func Forbidden(code, msg string) *Error    { return newErr(KindForbidden, code, msg) }
func Conflict(code, msg string) *Error     { return newErr(KindConflict, code, msg) }
func Unauthorized(code, msg string) *Error { return newErr(KindUnauthorized, code, msg) }
func Unavailable(code, msg string) *Error  { return newErr(KindUnavailable, code, msg) }
func RateLimited(code, msg string) *Error  { return newErr(KindRateLimited, code, msg) }

// ErrNotFound là lỗi "không tìm thấy" chung mà repository trả về.
var ErrNotFound = NotFound("not_found", "not found")

func KindOf(err error) ErrorKind {
	var de *Error
	if errors.As(err, &de) {
		return de.Kind
	}
	return KindInternal
}

func IsNotFound(err error) bool { return KindOf(err) == KindNotFound }
