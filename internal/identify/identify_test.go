package identify

import (
	"context"
	"errors"
	"reflect"
	"testing"

	"github.com/stashapp/stash/pkg/models"
	"github.com/stashapp/stash/pkg/models/mocks"
	"github.com/stashapp/stash/pkg/scraper"
	"github.com/stashapp/stash/pkg/sliceutil/intslice"
	"github.com/stretchr/testify/mock"
)

var testCtx = context.Background()

type mockSceneScraper struct {
	errIDs  []int
	results map[int]*scraper.ScrapedScene
}

func (s mockSceneScraper) ScrapeScene(ctx context.Context, sceneID int) (*scraper.ScrapedScene, error) {
	if intslice.IntInclude(s.errIDs, sceneID) {
		return nil, errors.New("scrape scene error")
	}
	return s.results[sceneID], nil
}

type mockHookExecutor struct {
}

func (s mockHookExecutor) ExecuteSceneUpdatePostHooks(ctx context.Context, input models.SceneUpdateInput, inputFields []string) {
}

func TestSceneIdentifier_Identify(t *testing.T) {
	const (
		errID1 = iota
		errID2
		missingID
		found1ID
		found2ID
		errUpdateID
	)

	var scrapedTitle = "scrapedTitle"

	defaultOptions := &MetadataOptions{}
	sources := []ScraperSource{
		{
			Scraper: mockSceneScraper{
				errIDs: []int{errID1},
				results: map[int]*scraper.ScrapedScene{
					found1ID: {
						Title: &scrapedTitle,
					},
				},
			},
		},
		{
			Scraper: mockSceneScraper{
				errIDs: []int{errID2},
				results: map[int]*scraper.ScrapedScene{
					found2ID: {
						Title: &scrapedTitle,
					},
					errUpdateID: {
						Title: &scrapedTitle,
					},
				},
			},
		},
	}

	mockSceneReaderWriter := &mocks.SceneReaderWriter{}

	mockSceneReaderWriter.On("UpdatePartial", mock.Anything, mock.MatchedBy(func(id int) bool {
		return id == errUpdateID
	}), mock.Anything).Return(nil, errors.New("update error"))
	mockSceneReaderWriter.On("UpdatePartial", mock.Anything, mock.MatchedBy(func(id int) bool {
		return id != errUpdateID
	}), mock.Anything).Return(nil, nil)

	tests := []struct {
		name    string
		sceneID int
		wantErr bool
	}{
		{
			"error scraping",
			errID1,
			false,
		},
		{
			"error scraping from second",
			errID2,
			false,
		},
		{
			"found in first scraper",
			found1ID,
			false,
		},
		{
			"found in second scraper",
			found2ID,
			false,
		},
		{
			"not found",
			missingID,
			false,
		},
		{
			"error modifying",
			errUpdateID,
			true,
		},
	}

	identifier := SceneIdentifier{
		SceneReaderUpdater:          mockSceneReaderWriter,
		DefaultOptions:              defaultOptions,
		Sources:                     sources,
		SceneUpdatePostHookExecutor: mockHookExecutor{},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			scene := &models.Scene{
				ID:           tt.sceneID,
				PerformerIDs: models.NewRelatedIDs([]int{}),
				TagIDs:       models.NewRelatedIDs([]int{}),
				StashIDs:     models.NewRelatedStashIDs([]models.StashID{}),
			}
			if err := identifier.Identify(testCtx, &mocks.TxnManager{}, scene); (err != nil) != tt.wantErr {
				t.Errorf("SceneIdentifier.Identify() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestSceneIdentifier_modifyScene(t *testing.T) {
	repo := models.Repository{
		TxnManager: &mocks.TxnManager{},
	}
	tr := &SceneIdentifier{}

	type args struct {
		scene  *models.Scene
		result *scrapeResult
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			"empty update",
			args{
				&models.Scene{
					PerformerIDs: models.NewRelatedIDs([]int{}),
					TagIDs:       models.NewRelatedIDs([]int{}),
					StashIDs:     models.NewRelatedStashIDs([]models.StashID{}),
				},
				&scrapeResult{
					result: &scraper.ScrapedScene{},
				},
			},
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := tr.modifyScene(testCtx, repo, tt.args.scene, tt.args.result); (err != nil) != tt.wantErr {
				t.Errorf("SceneIdentifier.modifyScene() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func Test_getFieldOptions(t *testing.T) {
	const (
		inFirst  = "inFirst"
		inSecond = "inSecond"
		inBoth   = "inBoth"
	)

	type args struct {
		options []MetadataOptions
	}
	tests := []struct {
		name string
		args args
		want map[string]*FieldOptions
	}{
		{
			"simple",
			args{
				[]MetadataOptions{
					{
						FieldOptions: []*FieldOptions{
							{
								Field:    inFirst,
								Strategy: FieldStrategyIgnore,
							},
							{
								Field:    inBoth,
								Strategy: FieldStrategyIgnore,
							},
						},
					},
					{
						FieldOptions: []*FieldOptions{
							{
								Field:    inSecond,
								Strategy: FieldStrategyMerge,
							},
							{
								Field:    inBoth,
								Strategy: FieldStrategyMerge,
							},
						},
					},
				},
			},
			map[string]*FieldOptions{
				inFirst: {
					Field:    inFirst,
					Strategy: FieldStrategyIgnore,
				},
				inSecond: {
					Field:    inSecond,
					Strategy: FieldStrategyMerge,
				},
				inBoth: {
					Field:    inBoth,
					Strategy: FieldStrategyIgnore,
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := getFieldOptions(tt.args.options); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("getFieldOptions() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_getScenePartial(t *testing.T) {
	var (
		originalTitle   = "originalTitle"
		originalDate    = "2001-01-01"
		originalDetails = "originalDetails"
		originalURL     = "originalURL"
	)

	var (
		scrapedTitle   = "scrapedTitle"
		scrapedDate    = "2002-02-02"
		scrapedDetails = "scrapedDetails"
		scrapedURL     = "scrapedURL"
	)

	originalDateObj := models.NewDate(originalDate)
	scrapedDateObj := models.NewDate(scrapedDate)

	originalScene := &models.Scene{
		Title:   originalTitle,
		Date:    &originalDateObj,
		Details: originalDetails,
		URL:     originalURL,
	}

	organisedScene := *originalScene
	organisedScene.Organized = true

	emptyScene := &models.Scene{}

	postPartial := models.ScenePartial{
		Title:   models.NewOptionalString(scrapedTitle),
		Date:    models.NewOptionalDate(scrapedDateObj),
		Details: models.NewOptionalString(scrapedDetails),
		URL:     models.NewOptionalString(scrapedURL),
	}

	scrapedScene := &scraper.ScrapedScene{
		Title:   &scrapedTitle,
		Date:    &scrapedDate,
		Details: &scrapedDetails,
		URL:     &scrapedURL,
	}

	scrapedUnchangedScene := &scraper.ScrapedScene{
		Title:   &originalTitle,
		Date:    &originalDate,
		Details: &originalDetails,
		URL:     &originalURL,
	}

	makeFieldOptions := func(input *FieldOptions) map[string]*FieldOptions {
		return map[string]*FieldOptions{
			"title":   input,
			"date":    input,
			"details": input,
			"url":     input,
		}
	}

	overwriteAll := makeFieldOptions(&FieldOptions{
		Strategy: FieldStrategyOverwrite,
	})
	ignoreAll := makeFieldOptions(&FieldOptions{
		Strategy: FieldStrategyIgnore,
	})
	mergeAll := makeFieldOptions(&FieldOptions{
		Strategy: FieldStrategyMerge,
	})

	setOrganised := true

	type args struct {
		scene        *models.Scene
		scraped      *scraper.ScrapedScene
		fieldOptions map[string]*FieldOptions
		setOrganized bool
	}
	tests := []struct {
		name string
		args args
		want models.ScenePartial
	}{
		{
			"overwrite all",
			args{
				originalScene,
				scrapedScene,
				overwriteAll,
				false,
			},
			postPartial,
		},
		{
			"ignore all",
			args{
				originalScene,
				scrapedScene,
				ignoreAll,
				false,
			},
			models.ScenePartial{},
		},
		{
			"merge (existing values)",
			args{
				originalScene,
				scrapedScene,
				mergeAll,
				false,
			},
			models.ScenePartial{},
		},
		{
			"merge (empty values)",
			args{
				emptyScene,
				scrapedScene,
				mergeAll,
				false,
			},
			postPartial,
		},
		{
			"unchanged",
			args{
				originalScene,
				scrapedUnchangedScene,
				overwriteAll,
				false,
			},
			models.ScenePartial{},
		},
		{
			"set organized",
			args{
				originalScene,
				scrapedUnchangedScene,
				overwriteAll,
				true,
			},
			models.ScenePartial{
				Organized: models.NewOptionalBool(setOrganised),
			},
		},
		{
			"set organized unchanged",
			args{
				&organisedScene,
				scrapedUnchangedScene,
				overwriteAll,
				true,
			},
			models.ScenePartial{},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := getScenePartial(tt.args.scene, tt.args.scraped, tt.args.fieldOptions, tt.args.setOrganized); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("getScenePartial() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_shouldSetSingleValueField(t *testing.T) {
	const invalid = "invalid"

	type args struct {
		strategy         *FieldOptions
		hasExistingValue bool
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			"ignore",
			args{
				&FieldOptions{
					Strategy: FieldStrategyIgnore,
				},
				false,
			},
			false,
		},
		{
			"merge existing",
			args{
				&FieldOptions{
					Strategy: FieldStrategyMerge,
				},
				true,
			},
			false,
		},
		{
			"merge absent",
			args{
				&FieldOptions{
					Strategy: FieldStrategyMerge,
				},
				false,
			},
			true,
		},
		{
			"overwrite",
			args{
				&FieldOptions{
					Strategy: FieldStrategyOverwrite,
				},
				true,
			},
			true,
		},
		{
			"nil (merge) existing",
			args{
				&FieldOptions{},
				true,
			},
			false,
		},
		{
			"nil (merge) absent",
			args{
				&FieldOptions{},
				false,
			},
			true,
		},
		{
			"invalid (merge) existing",
			args{
				&FieldOptions{
					Strategy: invalid,
				},
				true,
			},
			false,
		},
		{
			"invalid (merge) absent",
			args{
				&FieldOptions{
					Strategy: invalid,
				},
				false,
			},
			true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := shouldSetSingleValueField(tt.args.strategy, tt.args.hasExistingValue); got != tt.want {
				t.Errorf("shouldSetSingleValueField() = %v, want %v", got, tt.want)
			}
		})
	}
}
