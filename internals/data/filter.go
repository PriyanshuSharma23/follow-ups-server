package data

import (
	"math"
	"strings"

	"github.com/PriyanshuSharma23/follow-ups-server/internals/validator"
)

type Filters struct {
	Sort         string
	SortSafelist []string
	Page         int
	PageSize     int
}

func ValidateFilter(v *validator.Validator, f Filters) {
	v.Check(validator.Min(f.Page, 1), "page", "must be greater than or equal to 1")
	v.Check(validator.Max(f.Page, 10_000_000), "page", "must be less than or equal to 10 million")

	v.Check(validator.Min(f.PageSize, 1), "page_size", "must be greater than or equal to 1")
	v.Check(validator.Max(f.PageSize, 100), "page_size", "must be less than or equal to 100")

	v.Check(validator.In(f.Sort, f.SortSafelist...), "sort", "invalid sort value")
}

func (f *Filters) sortColumn() string {
	for _, safeVal := range f.SortSafelist {
		if safeVal == f.Sort {
			return strings.TrimPrefix(safeVal, "-")
		}
	}

	panic("unsafe sort value: " + f.Sort)
}

func (f *Filters) sortDirection() string {
	if strings.HasPrefix(f.Sort, "-") {
		return "DESC"
	}
	return "ASC"
}

func (f *Filters) limit() int {
	return f.PageSize
}

func (f *Filters) offset() int {
	return (f.Page - 1) * f.PageSize
}

type Metadata struct {
	CurrentPage  int `json:"current_page,omitempty"`
	PageSize     int `json:"page_size,omitempty"`
	FirstPage    int `json:"first_page,omitempty"`
	LastPage     int `json:"last_page,omitempty"`
	TotalRecords int `json:"total_records,omitempty"`
}

func calculateMetadata(totalRecords, page, pageSize int) Metadata {
	if totalRecords == 0 {
		return Metadata{}
	}

	lastPage := math.Ceil(float64(totalRecords) / float64(pageSize))

	return Metadata{
		CurrentPage:  page,
		PageSize:     pageSize,
		FirstPage:    1,
		LastPage:     int(lastPage),
		TotalRecords: totalRecords,
	}
}
