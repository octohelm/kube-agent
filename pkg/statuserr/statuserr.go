package statuserr

import (
	"encoding/json"
	"fmt"
	"net/http"
)

func New(code int, err error) *StatusErr {
	if e, ok := err.(*StatusErr); ok {
		return e
	}
	msg := ""
	if err != nil {
		msg = err.Error()
	}
	return &StatusErr{Code: code, Msg: msg, err: err}
}

type StatusErr struct {
	Code int    `json:"code"`
	Msg  string `json:"msg"`

	err error
}

func (se *StatusErr) Error() string {
	return fmt.Sprintf("[%d] %+v", se.Code, se.err)
}

func WriteToResp(rw http.ResponseWriter, se *StatusErr) {
	rw.Header().Set("Content-Type", "application/json; charset=utf-8")
	rw.WriteHeader(se.Code)
	_ = json.NewEncoder(rw).Encode(se)
}
