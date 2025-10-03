package repo

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo/integration/mtest"
)

func TestListWithCursorAndFilters_SearchCombinedWithClubFilters(t *testing.T) {
	mt := mtest.New(t, mtest.NewOptions().ClientType(mtest.Mock))

	mt.Run("search query works with club filter", func(mt *mtest.T) {
		repo := &PlayerRepo{c: mt.Coll}

		namespace := mt.Coll.Database().Name() + "." + mt.Coll.Name()
		mt.AddMockResponses(mtest.CreateCursorResponse(0, namespace, mtest.FirstBatch))

		clubID := primitive.NewObjectID()
		filters := PlayerListFilters{
			SearchQuery: "anna",
			ClubIDs:     []string{clubID.Hex()},
		}

		_, _, _, _, _, err := repo.ListWithCursorAndFilters(context.Background(), 5, "", "", filters)
		require.NoError(mt, err)

		started := mt.GetStartedEvent()
		require.NotNil(mt, started)

		filterValue := started.Command.Lookup("filter")

		filterDoc, ok := filterValue.DocumentOK()
		require.True(mt, ok)

		var filterMap bson.M
		err = bson.Unmarshal(filterDoc, &filterMap)
		require.NoError(mt, err)

		andSlice, ok := filterMap["$and"].(bson.A)
		require.True(mt, ok, "expected $and clause combining search and club filters")

		var foundSearch, foundClub bool

		for _, conditionAny := range andSlice {
			condition, ok := conditionAny.(bson.M)
			require.True(mt, ok)

			if orSlice, ok := condition["$or"].(bson.A); ok {
				for _, orAny := range orSlice {
					orCondition, ok := orAny.(bson.M)
					require.True(mt, ok)
					if display, ok := orCondition["display_name"].(bson.M); ok {
						if regex, ok := display["$regex"].(string); ok && regex == filters.SearchQuery {
							foundSearch = true
						}
					}
				}
			}

			if clubCond, ok := condition["club_memberships"].(bson.M); ok {
				elemMatch, ok := clubCond["$elemMatch"].(bson.M)
				require.True(mt, ok)

				clubIDCond, ok := elemMatch["club_id"].(bson.M)
				require.True(mt, ok)

				inValues, ok := clubIDCond["$in"].(bson.A)
				require.True(mt, ok)
				require.Contains(mt, inValues, clubID)
				foundClub = true
			}
		}

		require.True(mt, foundSearch, "expected search condition to remain when club filter is applied")
		require.True(mt, foundClub, "expected club filter to be present in the query")
	})
}
