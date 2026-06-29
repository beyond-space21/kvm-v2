package httpx

import (
	"net/http"
	"strconv"
)

type PageParams struct {
	Page  int
	Limit int
	Offset int
}

func ParsePagination(r *http.Request) PageParams {
	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))

	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 20
	}

	return PageParams{
		Page:   page,
		Limit:  limit,
		Offset: (page - 1) * limit,
	}
}
