package cmd

import (
	"github.com/sirupsen/logrus"
)

type baseResponse struct {
	Success  bool   `json:"success"`
	ErrorMsg string `json:"errorMsg"`
}

type importerResponse struct {
	Name          string `json:"name"`
	AlreadyExists bool   `json:"alreadyExists"`
	baseResponse  `json:",inline"`
}

// ToLogrusFields is a helper for the logrus library
func (i importerResponse) ToLogrusFields() logrus.Fields {
	return logrus.Fields{
		"success":       i.Success,
		"errorMsg":      i.ErrorMsg,
		"name":          i.Name,
		"alreadyExists": i.AlreadyExists,
	}
}
