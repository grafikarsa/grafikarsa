package http

import (
	"context"
	"encoding/json"
	"math"

	"github.com/jackc/pgx/v5/pgtype"
	"grafikarsa/api/internal/repo"
)

func trackLikeInterest(ctx context.Context, a *App, userID string, portfolioID pgtype.UUID) {
	tags, _ := a.Q.ListPortfolioTags(ctx, portfolioID)
	owner, err := a.Q.GetPortfolioByID(ctx, portfolioID)
	if err != nil {
		return
	}
	ownerFull, err := a.Q.GetUserByID(ctx, owner.OwnerID)
	if err != nil {
		return
	}
	interest, _ := a.Q.GetUserInterest(ctx, mustUUID(userID))
	tagW := map[string]float64{}
	majorW := map[string]float64{}
	total := 0
	if interest.UserID.Valid {
		_ = json.Unmarshal(interest.TagWeights, &tagW)
		_ = json.Unmarshal(interest.MajorWeights, &majorW)
		total = int(interest.TotalLikes)
	}
	for _, t := range tags {
		tagW[uuidStr(t.ID)]++
	}
	if ownerFull.MajorID.Valid {
		majorW[uuidStr(ownerFull.MajorID)]++
	}
	total++
	tw, _ := json.Marshal(tagW)
	mw, _ := json.Marshal(majorW)
	_ = a.Q.UpsertUserInterest(ctx, repo.UpsertUserInterestParams{
		UserID: mustUUID(userID), TagWeights: tw, MajorWeights: mw, TotalLikes: int32(total),
	})
}

func cosineSimilarity(a, b map[string]float64, keys []string) float64 {
	var dot, na, nb float64
	for _, k := range keys {
		dot += a[k] * b[k]
		na += a[k] * a[k]
		nb += b[k] * b[k]
	}
	if na == 0 || nb == 0 {
		return 0
	}
	return dot / (sqrt(na) * sqrt(nb))
}

func sqrt(f float64) float64 {
	return math.Sqrt(f)
}

func log1p(f float64) float64 {
	return math.Log1p(f)
}
