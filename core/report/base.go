/*
*
*	Ddosify - Load testing tool for any web system.
*   Copyright (C) 2021  Ddosify (https://ddosify.com)
*
*   This program is free software: you can redistribute it and/or modify
*   it under the terms of the GNU Affero General Public License as published
*   by the Free Software Foundation, either version 3 of the License, or
*   (at your option) any later version.
*
*   This program is distributed in the hope that it will be useful,
*   but WITHOUT ANY WARRANTY; without even the implied warranty of
*   MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
*   GNU Affero General Public License for more details.
*
*   You should have received a copy of the GNU Affero General Public License
*   along with this program.  If not, see <https://www.gnu.org/licenses/>.
*
 */

package report

import (
	"fmt"
	"strings"

	"ddosify.com/hammer/core/types"
)

// ReportService is the interface that abstracts different report implementations.
type ReportService interface {
	DoneChan() <-chan struct{}
	Init() error
	Start(input chan *types.Response)
	Report()
}

// Factory method of the ProxyService.
// Available strategies are in types.SupportedOutputs.
func NewReportService(s string) (service ReportService, err error) {
	if strings.EqualFold(s, types.OutputTypeStdout) {
		service = &stdout{}
	} else if strings.EqualFold(s, types.OutputTypeTimescale) {
		service = &timescale{}
	} else {
		err = fmt.Errorf("unsupported output type")
	}
	return
}
