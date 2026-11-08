package api

import (
	"context"

	"github.com/grafikarsa/grafikarsa/apps/api/internal/domain/taxonomy"
	"github.com/grafikarsa/grafikarsa/apps/api/internal/http/apigen"
)

// GetTaxonomyCategories lists the active categories of a content type.
func (s *Server) GetTaxonomyCategories(ctx context.Context, request apigen.GetTaxonomyCategoriesRequestObject) (apigen.GetTaxonomyCategoriesResponseObject, error) {
	categories, err := s.taxonomy.Categories(ctx, string(request.Params.Type))
	if err != nil {
		return nil, err
	}

	items := make([]apigen.CategoryOption, 0, len(categories))
	for _, category := range categories {
		majorCodes := category.MajorCodes
		if majorCodes == nil {
			majorCodes = []string{}
		}
		items = append(items, apigen.CategoryOption{
			Id:          category.ID,
			Slug:        category.Slug,
			Name:        category.Name,
			Description: category.Description,
			MajorCodes:  majorCodes,
		})
	}

	return apigen.GetTaxonomyCategories200JSONResponse{Items: items}, nil
}

// GetTaxonomyTools searches tools for the content editor autocomplete.
func (s *Server) GetTaxonomyTools(ctx context.Context, request apigen.GetTaxonomyToolsRequestObject) (apigen.GetTaxonomyToolsResponseObject, error) {
	terms, err := s.taxonomy.Tools(ctx, queryParam(request.Params.Q))
	if err != nil {
		return nil, err
	}

	return apigen.GetTaxonomyTools200JSONResponse{Items: toTaxonomyTerms(terms)}, nil
}

// GetTaxonomyTags searches tags for the content editor autocomplete.
func (s *Server) GetTaxonomyTags(ctx context.Context, request apigen.GetTaxonomyTagsRequestObject) (apigen.GetTaxonomyTagsResponseObject, error) {
	terms, err := s.taxonomy.Tags(ctx, queryParam(request.Params.Q))
	if err != nil {
		return nil, err
	}

	return apigen.GetTaxonomyTags200JSONResponse{Items: toTaxonomyTerms(terms)}, nil
}

func queryParam(q *string) string {
	if q == nil {
		return ""
	}
	return *q
}

func toTaxonomyTerms(terms []taxonomy.Term) []apigen.TaxonomyTerm {
	items := make([]apigen.TaxonomyTerm, 0, len(terms))
	for _, term := range terms {
		items = append(items, apigen.TaxonomyTerm{
			Slug:     term.Slug,
			Name:     term.Name,
			Category: term.Category,
		})
	}
	return items
}
