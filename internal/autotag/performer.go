package autotag

import (
	"context"

	"github.com/stashapp/stash/pkg/gallery"
	"github.com/stashapp/stash/pkg/image"
	"github.com/stashapp/stash/pkg/match"
	"github.com/stashapp/stash/pkg/models"
	"github.com/stashapp/stash/pkg/scene"
	"github.com/stashapp/stash/pkg/sliceutil/intslice"
	"github.com/stashapp/stash/pkg/txn"
)

type SceneQueryPerformerUpdater interface {
	scene.Queryer
	models.PerformerIDLoader
	scene.PartialUpdater
}

type ImageQueryPerformerUpdater interface {
	image.Queryer
	models.PerformerIDLoader
	image.PartialUpdater
}

type GalleryQueryPerformerUpdater interface {
	gallery.Queryer
	models.PerformerIDLoader
	gallery.PartialUpdater
}

func getPerformerTagger(p *models.Performer, cache *match.Cache) tagger {
	return tagger{
		ID:    p.ID,
		Type:  "performer",
		Name:  p.Name,
		cache: cache,
	}
}

// PerformerScenes searches for scenes whose path matches the provided performer name and tags the scene with the performer.
func (tagger *Tagger) PerformerScenes(ctx context.Context, p *models.Performer, paths []string, rw SceneQueryPerformerUpdater) error {
	t := getPerformerTagger(p, tagger.Cache)

	return t.tagScenes(ctx, paths, rw, func(o *models.Scene) (bool, error) {
		if err := o.LoadPerformerIDs(ctx, rw); err != nil {
			return false, err
		}
		existing := o.PerformerIDs.List()

		if intslice.IntInclude(existing, p.ID) {
			return false, nil
		}

		if err := txn.WithTxn(ctx, tagger.TxnManager, func(ctx context.Context) error {
			return scene.AddPerformer(ctx, rw, o, p.ID)
		}); err != nil {
			return false, err
		}

		return true, nil
	})
}

// PerformerImages searches for images whose path matches the provided performer name and tags the image with the performer.
func (tagger *Tagger) PerformerImages(ctx context.Context, p *models.Performer, paths []string, rw ImageQueryPerformerUpdater) error {
	t := getPerformerTagger(p, tagger.Cache)

	return t.tagImages(ctx, paths, rw, func(o *models.Image) (bool, error) {
		if err := o.LoadPerformerIDs(ctx, rw); err != nil {
			return false, err
		}
		existing := o.PerformerIDs.List()

		if intslice.IntInclude(existing, p.ID) {
			return false, nil
		}

		if err := txn.WithTxn(ctx, tagger.TxnManager, func(ctx context.Context) error {
			return image.AddPerformer(ctx, rw, o, p.ID)
		}); err != nil {
			return false, err
		}

		return true, nil
	})
}

// PerformerGalleries searches for galleries whose path matches the provided performer name and tags the gallery with the performer.
func (tagger *Tagger) PerformerGalleries(ctx context.Context, p *models.Performer, paths []string, rw GalleryQueryPerformerUpdater) error {
	t := getPerformerTagger(p, tagger.Cache)

	return t.tagGalleries(ctx, paths, rw, func(o *models.Gallery) (bool, error) {
		if err := o.LoadPerformerIDs(ctx, rw); err != nil {
			return false, err
		}
		existing := o.PerformerIDs.List()

		if intslice.IntInclude(existing, p.ID) {
			return false, nil
		}

		if err := txn.WithTxn(ctx, tagger.TxnManager, func(ctx context.Context) error {
			return gallery.AddPerformer(ctx, rw, o, p.ID)
		}); err != nil {
			return false, err
		}

		return true, nil
	})
}
