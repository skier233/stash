package models

import "context"

type ImageFilterType struct {
	And   *ImageFilterType      `json:"AND"`
	Or    *ImageFilterType      `json:"OR"`
	Not   *ImageFilterType      `json:"NOT"`
	ID    *IntCriterionInput    `json:"id"`
	Title *StringCriterionInput `json:"title"`
	// Filter by file checksum
	Checksum *StringCriterionInput `json:"checksum"`
	// Filter by path
	Path *StringCriterionInput `json:"path"`
	// Filter by file count
	FileCount *IntCriterionInput `json:"file_count"`
	// Filter by rating expressed as 1-5
	Rating *IntCriterionInput `json:"rating"`
	// Filter by rating expressed as 1-100
	Rating100 *IntCriterionInput `json:"rating100"`
	// Filter by organized
	Organized *bool `json:"organized"`
	// Filter by o-counter
	OCounter *IntCriterionInput `json:"o_counter"`
	// Filter by resolution
	Resolution *ResolutionCriterionInput `json:"resolution"`
	// Filter to only include images missing this property
	IsMissing *string `json:"is_missing"`
	// Filter to only include images with this studio
	Studios *HierarchicalMultiCriterionInput `json:"studios"`
	// Filter to only include images with these tags
	Tags *HierarchicalMultiCriterionInput `json:"tags"`
	// Filter by tag count
	TagCount *IntCriterionInput `json:"tag_count"`
	// Filter to only include images with performers with these tags
	PerformerTags *HierarchicalMultiCriterionInput `json:"performer_tags"`
	// Filter to only include images with these performers
	Performers *MultiCriterionInput `json:"performers"`
	// Filter by performer count
	PerformerCount *IntCriterionInput `json:"performer_count"`
	// Filter images that have performers that have been favorited
	PerformerFavorite *bool `json:"performer_favorite"`
	// Filter to only include images with these galleries
	Galleries *MultiCriterionInput `json:"galleries"`
	// Filter by created at
	CreatedAt *TimestampCriterionInput `json:"created_at"`
	// Filter by updated at
	UpdatedAt *TimestampCriterionInput `json:"updated_at"`
}

type ImageDestroyInput struct {
	ID              string `json:"id"`
	DeleteFile      *bool  `json:"delete_file"`
	DeleteGenerated *bool  `json:"delete_generated"`
}

type ImagesDestroyInput struct {
	Ids             []string `json:"ids"`
	DeleteFile      *bool    `json:"delete_file"`
	DeleteGenerated *bool    `json:"delete_generated"`
}

type ImageQueryOptions struct {
	QueryOptions
	ImageFilter *ImageFilterType

	Megapixels bool
	TotalSize  bool
}

type ImageQueryResult struct {
	QueryResult
	Megapixels float64
	TotalSize  float64

	finder     ImageFinder
	images     []*Image
	resolveErr error
}

func NewImageQueryResult(finder ImageFinder) *ImageQueryResult {
	return &ImageQueryResult{
		finder: finder,
	}
}

func (r *ImageQueryResult) Resolve(ctx context.Context) ([]*Image, error) {
	// cache results
	if r.images == nil && r.resolveErr == nil {
		r.images, r.resolveErr = r.finder.FindMany(ctx, r.IDs)
	}
	return r.images, r.resolveErr
}

type ImageFinder interface {
	// TODO - rename to Find and remove existing method
	FindMany(ctx context.Context, ids []int) ([]*Image, error)
}

type ImageReader interface {
	ImageFinder
	// TODO - remove this in another PR
	Find(ctx context.Context, id int) (*Image, error)
	FindByChecksum(ctx context.Context, checksum string) ([]*Image, error)
	FindByGalleryID(ctx context.Context, galleryID int) ([]*Image, error)
	CountByGalleryID(ctx context.Context, galleryID int) (int, error)
	Count(ctx context.Context) (int, error)
	Size(ctx context.Context) (float64, error)
	All(ctx context.Context) ([]*Image, error)
	Query(ctx context.Context, options ImageQueryOptions) (*ImageQueryResult, error)
	QueryCount(ctx context.Context, imageFilter *ImageFilterType, findFilter *FindFilterType) (int, error)

	GalleryIDLoader
	PerformerIDLoader
	TagIDLoader
}

type ImageWriter interface {
	Create(ctx context.Context, newImage *ImageCreateInput) error
	Update(ctx context.Context, updatedImage *Image) error
	UpdatePartial(ctx context.Context, id int, partial ImagePartial) (*Image, error)
	IncrementOCounter(ctx context.Context, id int) (int, error)
	DecrementOCounter(ctx context.Context, id int) (int, error)
	ResetOCounter(ctx context.Context, id int) (int, error)
	Destroy(ctx context.Context, id int) error
}

type ImageReaderWriter interface {
	ImageReader
	ImageWriter
}
