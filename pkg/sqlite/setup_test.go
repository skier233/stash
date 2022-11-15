//go:build integration
// +build integration

package sqlite_test

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"testing"
	"time"

	"github.com/stashapp/stash/pkg/file"
	"github.com/stashapp/stash/pkg/hash/md5"
	"github.com/stashapp/stash/pkg/models"
	"github.com/stashapp/stash/pkg/sliceutil/intslice"
	"github.com/stashapp/stash/pkg/sqlite"
	"github.com/stashapp/stash/pkg/txn"

	// necessary to register custom migrations
	_ "github.com/stashapp/stash/pkg/sqlite/migrations"
)

const (
	spacedSceneTitle = "zzz yyy xxx"
)

const (
	folderIdxWithSubFolder = iota
	folderIdxWithParentFolder
	folderIdxWithFiles
	folderIdxInZip

	folderIdxForObjectFiles
	folderIdxWithImageFiles
	folderIdxWithGalleryFiles
	folderIdxWithSceneFiles

	totalFolders
)

const (
	fileIdxZip = iota
	fileIdxInZip

	fileIdxStartVideoFiles
	fileIdxStartImageFiles
	fileIdxStartGalleryFiles

	totalFiles
)

const (
	sceneIdxWithMovie = iota
	sceneIdxWithGallery
	sceneIdxWithPerformer
	sceneIdx1WithPerformer
	sceneIdx2WithPerformer
	sceneIdxWithTwoPerformers
	sceneIdxWithTag
	sceneIdxWithTwoTags
	sceneIdxWithMarkerAndTag
	sceneIdxWithStudio
	sceneIdx1WithStudio
	sceneIdx2WithStudio
	sceneIdxWithMarkers
	sceneIdxWithPerformerTag
	sceneIdxWithPerformerTwoTags
	sceneIdxWithSpacedName
	sceneIdxWithStudioPerformer
	sceneIdxWithGrandChildStudio
	sceneIdxMissingPhash
	// new indexes above
	lastSceneIdx

	totalScenes = lastSceneIdx + 3
)

const dupeScenePhashes = 2

const (
	imageIdxWithGallery = iota
	imageIdx1WithGallery
	imageIdx2WithGallery
	imageIdxWithTwoGalleries
	imageIdxWithPerformer
	imageIdx1WithPerformer
	imageIdx2WithPerformer
	imageIdxWithTwoPerformers
	imageIdxWithTag
	imageIdxWithTwoTags
	imageIdxWithStudio
	imageIdx1WithStudio
	imageIdx2WithStudio
	imageIdxWithStudioPerformer
	imageIdxInZip
	imageIdxWithPerformerTag
	imageIdxWithPerformerTwoTags
	imageIdxWithGrandChildStudio
	// new indexes above
	totalImages
)

const (
	performerIdxWithScene = iota
	performerIdx1WithScene
	performerIdx2WithScene
	performerIdxWithTwoScenes
	performerIdxWithImage
	performerIdxWithTwoImages
	performerIdx1WithImage
	performerIdx2WithImage
	performerIdxWithTag
	performerIdxWithTwoTags
	performerIdxWithGallery
	performerIdxWithTwoGalleries
	performerIdx1WithGallery
	performerIdx2WithGallery
	performerIdxWithSceneStudio
	performerIdxWithImageStudio
	performerIdxWithGalleryStudio
	// new indexes above
	// performers with dup names start from the end
	performerIdx1WithDupName
	performerIdxWithDupName

	performersNameCase   = performerIdx1WithDupName
	performersNameNoCase = 2

	totalPerformers = performersNameCase + performersNameNoCase
)

const (
	movieIdxWithScene = iota
	movieIdxWithStudio
	// movies with dup names start from the end
	// create 10 more basic movies (can remove this if we add more indexes)
	movieIdxWithDupName = movieIdxWithStudio + 10

	moviesNameCase   = movieIdxWithDupName
	moviesNameNoCase = 1
)

const (
	galleryIdxWithScene = iota
	galleryIdxWithImage
	galleryIdx1WithImage
	galleryIdx2WithImage
	galleryIdxWithTwoImages
	galleryIdxWithPerformer
	galleryIdx1WithPerformer
	galleryIdx2WithPerformer
	galleryIdxWithTwoPerformers
	galleryIdxWithTag
	galleryIdxWithTwoTags
	galleryIdxWithStudio
	galleryIdx1WithStudio
	galleryIdx2WithStudio
	galleryIdxWithPerformerTag
	galleryIdxWithPerformerTwoTags
	galleryIdxWithStudioPerformer
	galleryIdxWithGrandChildStudio
	galleryIdxWithoutFile
	// new indexes above
	lastGalleryIdx

	totalGalleries = lastGalleryIdx + 1
)

const (
	tagIdxWithScene = iota
	tagIdx1WithScene
	tagIdx2WithScene
	tagIdx3WithScene
	tagIdxWithPrimaryMarkers
	tagIdxWithMarkers
	tagIdxWithCoverImage
	tagIdxWithImage
	tagIdx1WithImage
	tagIdx2WithImage
	tagIdxWithPerformer
	tagIdx1WithPerformer
	tagIdx2WithPerformer
	tagIdxWithGallery
	tagIdx1WithGallery
	tagIdx2WithGallery
	tagIdxWithChildTag
	tagIdxWithParentTag
	tagIdxWithGrandChild
	tagIdxWithParentAndChild
	tagIdxWithGrandParent
	// new indexes above
	// tags with dup names start from the end
	tagIdx1WithDupName
	tagIdxWithDupName

	tagsNameNoCase = 2
	tagsNameCase   = tagIdx1WithDupName

	totalTags = tagsNameCase + tagsNameNoCase
)

const (
	studioIdxWithScene = iota
	studioIdxWithTwoScenes
	studioIdxWithMovie
	studioIdxWithChildStudio
	studioIdxWithParentStudio
	studioIdxWithImage
	studioIdxWithTwoImages
	studioIdxWithGallery
	studioIdxWithTwoGalleries
	studioIdxWithScenePerformer
	studioIdxWithImagePerformer
	studioIdxWithGalleryPerformer
	studioIdxWithGrandChild
	studioIdxWithParentAndChild
	studioIdxWithGrandParent
	// new indexes above
	// studios with dup names start from the end
	studioIdxWithDupName

	studiosNameCase   = studioIdxWithDupName
	studiosNameNoCase = 1

	totalStudios = studiosNameCase + studiosNameNoCase
)

const (
	markerIdxWithScene = iota
	markerIdxWithTag
	markerIdxWithSceneTag
	totalMarkers
)

const (
	savedFilterIdxDefaultScene = iota
	savedFilterIdxDefaultImage
	savedFilterIdxScene
	savedFilterIdxImage

	// new indexes above
	totalSavedFilters
)

const (
	pathField            = "Path"
	checksumField        = "Checksum"
	titleField           = "Title"
	urlField             = "URL"
	zipPath              = "zipPath.zip"
	firstSavedFilterName = "firstSavedFilterName"
)

var (
	folderIDs      []file.FolderID
	fileIDs        []file.ID
	sceneFileIDs   []file.ID
	imageFileIDs   []file.ID
	galleryFileIDs []file.ID

	sceneIDs       []int
	imageIDs       []int
	performerIDs   []int
	movieIDs       []int
	galleryIDs     []int
	tagIDs         []int
	studioIDs      []int
	markerIDs      []int
	savedFilterIDs []int

	folderPaths []string

	tagNames       []string
	studioNames    []string
	movieNames     []string
	performerNames []string
)

type idAssociation struct {
	first  int
	second int
}

type linkMap map[int][]int

func (m linkMap) reverseLookup(idx int) []int {
	var result []int

	for k, v := range m {
		for _, vv := range v {
			if vv == idx {
				result = append(result, k)
			}
		}
	}

	return result
}

var (
	folderParentFolders = map[int]int{
		folderIdxWithParentFolder: folderIdxWithSubFolder,
		folderIdxWithSceneFiles:   folderIdxForObjectFiles,
		folderIdxWithImageFiles:   folderIdxForObjectFiles,
		folderIdxWithGalleryFiles: folderIdxForObjectFiles,
	}

	fileFolders = map[int]int{
		fileIdxZip:   folderIdxWithFiles,
		fileIdxInZip: folderIdxInZip,
	}

	folderZipFiles = map[int]int{
		folderIdxInZip: fileIdxZip,
	}

	fileZipFiles = map[int]int{
		fileIdxInZip: fileIdxZip,
	}
)

var (
	sceneTags = linkMap{
		sceneIdxWithTag:          {tagIdxWithScene},
		sceneIdxWithTwoTags:      {tagIdx1WithScene, tagIdx2WithScene},
		sceneIdxWithMarkerAndTag: {tagIdx3WithScene},
	}

	scenePerformers = linkMap{
		sceneIdxWithPerformer:        {performerIdxWithScene},
		sceneIdxWithTwoPerformers:    {performerIdx1WithScene, performerIdx2WithScene},
		sceneIdxWithPerformerTag:     {performerIdxWithTag},
		sceneIdxWithPerformerTwoTags: {performerIdxWithTwoTags},
		sceneIdx1WithPerformer:       {performerIdxWithTwoScenes},
		sceneIdx2WithPerformer:       {performerIdxWithTwoScenes},
		sceneIdxWithStudioPerformer:  {performerIdxWithSceneStudio},
	}

	sceneGalleries = linkMap{
		sceneIdxWithGallery: {galleryIdxWithScene},
	}

	sceneMovies = linkMap{
		sceneIdxWithMovie: {movieIdxWithScene},
	}

	sceneStudios = map[int]int{
		sceneIdxWithStudio:           studioIdxWithScene,
		sceneIdx1WithStudio:          studioIdxWithTwoScenes,
		sceneIdx2WithStudio:          studioIdxWithTwoScenes,
		sceneIdxWithStudioPerformer:  studioIdxWithScenePerformer,
		sceneIdxWithGrandChildStudio: studioIdxWithGrandParent,
	}
)

type markerSpec struct {
	sceneIdx      int
	primaryTagIdx int
	tagIdxs       []int
}

var (
	// indexed by marker
	markerSpecs = []markerSpec{
		{sceneIdxWithMarkers, tagIdxWithPrimaryMarkers, nil},
		{sceneIdxWithMarkers, tagIdxWithPrimaryMarkers, []int{tagIdxWithMarkers}},
		{sceneIdxWithMarkerAndTag, tagIdxWithPrimaryMarkers, nil},
	}
)

var (
	imageGalleries = linkMap{
		imageIdxWithGallery:      {galleryIdxWithImage},
		imageIdx1WithGallery:     {galleryIdxWithTwoImages},
		imageIdx2WithGallery:     {galleryIdxWithTwoImages},
		imageIdxWithTwoGalleries: {galleryIdx1WithImage, galleryIdx2WithImage},
	}
	imageStudios = map[int]int{
		imageIdxWithStudio:           studioIdxWithImage,
		imageIdx1WithStudio:          studioIdxWithTwoImages,
		imageIdx2WithStudio:          studioIdxWithTwoImages,
		imageIdxWithStudioPerformer:  studioIdxWithImagePerformer,
		imageIdxWithGrandChildStudio: studioIdxWithGrandParent,
	}
	imageTags = linkMap{
		imageIdxWithTag:     {tagIdxWithImage},
		imageIdxWithTwoTags: {tagIdx1WithImage, tagIdx2WithImage},
	}
	imagePerformers = linkMap{
		imageIdxWithPerformer:        {performerIdxWithImage},
		imageIdxWithTwoPerformers:    {performerIdx1WithImage, performerIdx2WithImage},
		imageIdxWithPerformerTag:     {performerIdxWithTag},
		imageIdxWithPerformerTwoTags: {performerIdxWithTwoTags},
		imageIdx1WithPerformer:       {performerIdxWithTwoImages},
		imageIdx2WithPerformer:       {performerIdxWithTwoImages},
		imageIdxWithStudioPerformer:  {performerIdxWithImageStudio},
	}
)

var (
	galleryPerformers = linkMap{
		galleryIdxWithPerformer:        {performerIdxWithGallery},
		galleryIdxWithTwoPerformers:    {performerIdx1WithGallery, performerIdx2WithGallery},
		galleryIdxWithPerformerTag:     {performerIdxWithTag},
		galleryIdxWithPerformerTwoTags: {performerIdxWithTwoTags},
		galleryIdx1WithPerformer:       {performerIdxWithTwoGalleries},
		galleryIdx2WithPerformer:       {performerIdxWithTwoGalleries},
		galleryIdxWithStudioPerformer:  {performerIdxWithGalleryStudio},
	}

	galleryStudios = map[int]int{
		galleryIdxWithStudio:           studioIdxWithGallery,
		galleryIdx1WithStudio:          studioIdxWithTwoGalleries,
		galleryIdx2WithStudio:          studioIdxWithTwoGalleries,
		galleryIdxWithStudioPerformer:  studioIdxWithGalleryPerformer,
		galleryIdxWithGrandChildStudio: studioIdxWithGrandParent,
	}

	galleryTags = linkMap{
		galleryIdxWithTag:     {tagIdxWithGallery},
		galleryIdxWithTwoTags: {tagIdx1WithGallery, tagIdx2WithGallery},
	}
)

var (
	movieStudioLinks = [][2]int{
		{movieIdxWithStudio, studioIdxWithMovie},
	}
)

var (
	studioParentLinks = [][2]int{
		{studioIdxWithChildStudio, studioIdxWithParentStudio},
		{studioIdxWithGrandChild, studioIdxWithParentAndChild},
		{studioIdxWithParentAndChild, studioIdxWithGrandParent},
	}
)

var (
	performerTagLinks = [][2]int{
		{performerIdxWithTag, tagIdxWithPerformer},
		{performerIdxWithTwoTags, tagIdx1WithPerformer},
		{performerIdxWithTwoTags, tagIdx2WithPerformer},
	}
)

var (
	tagParentLinks = [][2]int{
		{tagIdxWithChildTag, tagIdxWithParentTag},
		{tagIdxWithGrandChild, tagIdxWithParentAndChild},
		{tagIdxWithParentAndChild, tagIdxWithGrandParent},
	}
)

func indexesToIDs(ids []int, indexes []int) []int {
	ret := make([]int, len(indexes))
	for i, idx := range indexes {
		ret[i] = ids[idx]
	}

	return ret
}

var db *sqlite.Database

func TestMain(m *testing.M) {
	ret := runTests(m)
	os.Exit(ret)
}

func withTxn(f func(ctx context.Context) error) error {
	return txn.WithTxn(context.Background(), db, f)
}

func withRollbackTxn(f func(ctx context.Context) error) error {
	var ret error
	withTxn(func(ctx context.Context) error {
		ret = f(ctx)
		return errors.New("fake error for rollback")
	})

	return ret
}

func runWithRollbackTxn(t *testing.T, name string, f func(t *testing.T, ctx context.Context)) {
	withRollbackTxn(func(ctx context.Context) error {
		t.Run(name, func(t *testing.T) {
			f(t, ctx)
		})
		return nil
	})
}

func testTeardown(databaseFile string) {
	err := db.Close()

	if err != nil {
		panic(err)
	}

	err = os.Remove(databaseFile)
	if err != nil {
		panic(err)
	}
}

func runTests(m *testing.M) int {
	// create the database file
	f, err := os.CreateTemp("", "*.sqlite")
	if err != nil {
		panic(fmt.Sprintf("Could not create temporary file: %s", err.Error()))
	}

	f.Close()
	databaseFile := f.Name()
	db = sqlite.NewDatabase()

	if err := db.Open(databaseFile); err != nil {
		panic(fmt.Sprintf("Could not initialize database: %s", err.Error()))
	}

	// defer close and delete the database
	defer testTeardown(databaseFile)

	err = populateDB()
	if err != nil {
		panic(fmt.Sprintf("Could not populate database: %s", err.Error()))
	} else {
		// run the tests
		return m.Run()
	}
}

func populateDB() error {
	if err := withTxn(func(ctx context.Context) error {
		if err := createFolders(ctx); err != nil {
			return fmt.Errorf("creating folders: %w", err)
		}

		if err := createFiles(ctx); err != nil {
			return fmt.Errorf("creating files: %w", err)
		}

		// TODO - link folders to zip files

		if err := createMovies(ctx, sqlite.MovieReaderWriter, moviesNameCase, moviesNameNoCase); err != nil {
			return fmt.Errorf("error creating movies: %s", err.Error())
		}

		if err := createPerformers(ctx, performersNameCase, performersNameNoCase); err != nil {
			return fmt.Errorf("error creating performers: %s", err.Error())
		}

		if err := createTags(ctx, sqlite.TagReaderWriter, tagsNameCase, tagsNameNoCase); err != nil {
			return fmt.Errorf("error creating tags: %s", err.Error())
		}

		if err := createStudios(ctx, sqlite.StudioReaderWriter, studiosNameCase, studiosNameNoCase); err != nil {
			return fmt.Errorf("error creating studios: %s", err.Error())
		}

		if err := createGalleries(ctx, totalGalleries); err != nil {
			return fmt.Errorf("error creating galleries: %s", err.Error())
		}

		if err := createScenes(ctx, totalScenes); err != nil {
			return fmt.Errorf("error creating scenes: %s", err.Error())
		}

		if err := createImages(ctx, totalImages); err != nil {
			return fmt.Errorf("error creating images: %s", err.Error())
		}

		if err := addTagImage(ctx, sqlite.TagReaderWriter, tagIdxWithCoverImage); err != nil {
			return fmt.Errorf("error adding tag image: %s", err.Error())
		}

		if err := createSavedFilters(ctx, sqlite.SavedFilterReaderWriter, totalSavedFilters); err != nil {
			return fmt.Errorf("error creating saved filters: %s", err.Error())
		}

		if err := linkPerformerTags(ctx); err != nil {
			return fmt.Errorf("error linking performer tags: %s", err.Error())
		}

		if err := linkMovieStudios(ctx, sqlite.MovieReaderWriter); err != nil {
			return fmt.Errorf("error linking movie studios: %s", err.Error())
		}

		if err := linkStudiosParent(ctx, sqlite.StudioReaderWriter); err != nil {
			return fmt.Errorf("error linking studios parent: %s", err.Error())
		}

		if err := linkTagsParent(ctx, sqlite.TagReaderWriter); err != nil {
			return fmt.Errorf("error linking tags parent: %s", err.Error())
		}

		for _, ms := range markerSpecs {
			if err := createMarker(ctx, sqlite.SceneMarkerReaderWriter, ms); err != nil {
				return fmt.Errorf("error creating scene marker: %s", err.Error())
			}
		}

		return nil
	}); err != nil {
		return err
	}

	return nil
}

func getFolderPath(index int, parentFolderIdx *int) string {
	path := getPrefixedStringValue("folder", index, pathField)

	if parentFolderIdx != nil {
		return filepath.Join(folderPaths[*parentFolderIdx], path)
	}

	return path
}

func getFolderModTime(index int) time.Time {
	return time.Date(2000, 1, (index%10)+1, 0, 0, 0, 0, time.UTC)
}

func makeFolder(i int) file.Folder {
	var folderID *file.FolderID
	var folderIdx *int
	if pidx, ok := folderParentFolders[i]; ok {
		folderIdx = &pidx
		v := folderIDs[pidx]
		folderID = &v
	}

	return file.Folder{
		ParentFolderID: folderID,
		DirEntry: file.DirEntry{
			// zip files have to be added after creating files
			ModTime: getFolderModTime(i),
		},
		Path: getFolderPath(i, folderIdx),
	}
}

func createFolders(ctx context.Context) error {
	qb := db.Folder

	for i := 0; i < totalFolders; i++ {
		folder := makeFolder(i)

		if err := qb.Create(ctx, &folder); err != nil {
			return fmt.Errorf("Error creating folder [%d] %v+: %s", i, folder, err.Error())
		}

		folderIDs = append(folderIDs, folder.ID)
		folderPaths = append(folderPaths, folder.Path)
	}

	return nil
}

func getFileBaseName(index int) string {
	return getPrefixedStringValue("file", index, "basename")
}

func getFileStringValue(index int, field string) string {
	return getPrefixedStringValue("file", index, field)
}

func getFileModTime(index int) time.Time {
	return getFolderModTime(index)
}

func getFileFingerprints(index int) []file.Fingerprint {
	return []file.Fingerprint{
		{
			Type:        "MD5",
			Fingerprint: getPrefixedStringValue("file", index, "md5"),
		},
		{
			Type:        "OSHASH",
			Fingerprint: getPrefixedStringValue("file", index, "oshash"),
		},
	}
}

func getFileSize(index int) int64 {
	return int64(index) * 10
}

func getFileDuration(index int) float64 {
	duration := (index % 4) + 1
	duration = duration * 100

	return float64(duration) + 0.432
}

func makeFile(i int) file.File {
	folderID := folderIDs[fileFolders[i]]
	if folderID == 0 {
		folderID = folderIDs[folderIdxWithFiles]
	}

	var zipFileID *file.ID
	if zipFileIndex, found := fileZipFiles[i]; found {
		zipFileID = &fileIDs[zipFileIndex]
	}

	var ret file.File
	baseFile := &file.BaseFile{
		Basename:       getFileBaseName(i),
		ParentFolderID: folderID,
		DirEntry: file.DirEntry{
			// zip files have to be added after creating files
			ModTime:   getFileModTime(i),
			ZipFileID: zipFileID,
		},
		Fingerprints: getFileFingerprints(i),
		Size:         getFileSize(i),
	}

	ret = baseFile

	if i >= fileIdxStartVideoFiles && i < fileIdxStartImageFiles {
		ret = &file.VideoFile{
			BaseFile:   baseFile,
			Format:     getFileStringValue(i, "format"),
			Width:      getWidth(i),
			Height:     getHeight(i),
			Duration:   getFileDuration(i),
			VideoCodec: getFileStringValue(i, "videoCodec"),
			AudioCodec: getFileStringValue(i, "audioCodec"),
			FrameRate:  getFileDuration(i) * 2,
			BitRate:    int64(getFileDuration(i)) * 3,
		}
	} else if i >= fileIdxStartImageFiles && i < fileIdxStartGalleryFiles {
		ret = &file.ImageFile{
			BaseFile: baseFile,
			Format:   getFileStringValue(i, "format"),
			Width:    getWidth(i),
			Height:   getHeight(i),
		}
	}

	return ret
}

func createFiles(ctx context.Context) error {
	qb := db.File

	for i := 0; i < totalFiles; i++ {
		file := makeFile(i)

		if err := qb.Create(ctx, file); err != nil {
			return fmt.Errorf("Error creating file [%d] %v+: %s", i, file, err.Error())
		}

		fileIDs = append(fileIDs, file.Base().ID)
	}

	return nil
}

func getPrefixedStringValue(prefix string, index int, field string) string {
	return fmt.Sprintf("%s_%04d_%s", prefix, index, field)
}

func getPrefixedNullStringValue(prefix string, index int, field string) sql.NullString {
	if index > 0 && index%5 == 0 {
		return sql.NullString{}
	}
	if index > 0 && index%6 == 0 {
		return sql.NullString{
			String: "",
			Valid:  true,
		}
	}
	return sql.NullString{
		String: getPrefixedStringValue(prefix, index, field),
		Valid:  true,
	}
}

func getSceneStringValue(index int, field string) string {
	return getPrefixedStringValue("scene", index, field)
}

func getScenePhash(index int, field string) int64 {
	return int64(index % (totalScenes - dupeScenePhashes) * 1234)
}

func getSceneStringPtr(index int, field string) *string {
	v := getPrefixedStringValue("scene", index, field)
	return &v
}

func getSceneNullStringPtr(index int, field string) *string {
	return getStringPtrFromNullString(getPrefixedNullStringValue("scene", index, field))
}

func getSceneEmptyString(index int, field string) string {
	v := getSceneNullStringPtr(index, field)
	if v == nil {
		return ""
	}

	return *v
}

func getSceneTitle(index int) string {
	switch index {
	case sceneIdxWithSpacedName:
		return spacedSceneTitle
	default:
		return getSceneStringValue(index, titleField)
	}
}

func getRating(index int) sql.NullInt64 {
	rating := index % 6
	return sql.NullInt64{Int64: int64(rating * 20), Valid: rating > 0}
}

func getIntPtr(r sql.NullInt64) *int {
	if !r.Valid {
		return nil
	}

	v := int(r.Int64)
	return &v
}

func getStringPtrFromNullString(r sql.NullString) *string {
	if !r.Valid || r.String == "" {
		return nil
	}

	v := r.String
	return &v
}

func getStringPtr(r string) *string {
	if r == "" {
		return nil
	}

	return &r
}

func getEmptyStringFromPtr(v *string) string {
	if v == nil {
		return ""
	}

	return *v
}

func getOCounter(index int) int {
	return index % 3
}

func getSceneDuration(index int) float64 {
	duration := index + 1
	duration = duration * 100

	return float64(duration) + 0.432
}

func getHeight(index int) int {
	heights := []int{200, 240, 300, 480, 700, 720, 800, 1080, 1500, 2160, 3000}
	height := heights[index%len(heights)]
	return height
}

func getWidth(index int) int {
	height := getHeight(index)
	return height * 2
}

func getObjectDate(index int) models.SQLiteDate {
	dates := []string{"null", "", "0001-01-01", "2001-02-03"}
	date := dates[index%len(dates)]
	return models.SQLiteDate{
		String: date,
		Valid:  date != "null",
	}
}

func getObjectDateObject(index int) *models.Date {
	d := getObjectDate(index)
	if !d.Valid {
		return nil
	}

	ret := models.NewDate(d.String)
	return &ret
}

func sceneStashID(i int) models.StashID {
	return models.StashID{
		StashID:  getSceneStringValue(i, "stashid"),
		Endpoint: getSceneStringValue(i, "endpoint"),
	}
}

func getSceneBasename(index int) string {
	return getSceneStringValue(index, pathField)
}

func makeSceneFile(i int) *file.VideoFile {
	fp := []file.Fingerprint{
		{
			Type:        file.FingerprintTypeMD5,
			Fingerprint: getSceneStringValue(i, checksumField),
		},
		{
			Type:        file.FingerprintTypeOshash,
			Fingerprint: getSceneStringValue(i, "oshash"),
		},
	}

	if i != sceneIdxMissingPhash {
		fp = append(fp, file.Fingerprint{
			Type:        file.FingerprintTypePhash,
			Fingerprint: getScenePhash(i, "phash"),
		})
	}

	return &file.VideoFile{
		BaseFile: &file.BaseFile{
			Path:           getFilePath(folderIdxWithSceneFiles, getSceneBasename(i)),
			Basename:       getSceneBasename(i),
			ParentFolderID: folderIDs[folderIdxWithSceneFiles],
			Fingerprints:   fp,
		},
		Duration: getSceneDuration(i),
		Height:   getHeight(i),
		Width:    getWidth(i),
	}
}

func makeScene(i int) *models.Scene {
	title := getSceneTitle(i)
	details := getSceneStringValue(i, "Details")

	var studioID *int
	if _, ok := sceneStudios[i]; ok {
		v := studioIDs[sceneStudios[i]]
		studioID = &v
	}

	gids := indexesToIDs(galleryIDs, sceneGalleries[i])
	pids := indexesToIDs(performerIDs, scenePerformers[i])
	tids := indexesToIDs(tagIDs, sceneTags[i])

	mids := indexesToIDs(movieIDs, sceneMovies[i])

	movies := make([]models.MoviesScenes, len(mids))
	for i, m := range mids {
		movies[i] = models.MoviesScenes{
			MovieID: m,
		}
	}

	rating := getRating(i)

	return &models.Scene{
		Title:        title,
		Details:      details,
		URL:          getSceneEmptyString(i, urlField),
		Rating:       getIntPtr(rating),
		OCounter:     getOCounter(i),
		Date:         getObjectDateObject(i),
		StudioID:     studioID,
		GalleryIDs:   models.NewRelatedIDs(gids),
		PerformerIDs: models.NewRelatedIDs(pids),
		TagIDs:       models.NewRelatedIDs(tids),
		Movies:       models.NewRelatedMovies(movies),
		StashIDs: models.NewRelatedStashIDs([]models.StashID{
			sceneStashID(i),
		}),
	}
}

func createScenes(ctx context.Context, n int) error {
	sqb := db.Scene
	fqb := db.File

	for i := 0; i < n; i++ {
		f := makeSceneFile(i)
		if err := fqb.Create(ctx, f); err != nil {
			return fmt.Errorf("creating scene file: %w", err)
		}
		sceneFileIDs = append(sceneFileIDs, f.ID)

		scene := makeScene(i)

		if err := sqb.Create(ctx, scene, []file.ID{f.ID}); err != nil {
			return fmt.Errorf("Error creating scene %v+: %s", scene, err.Error())
		}

		sceneIDs = append(sceneIDs, scene.ID)
	}

	return nil
}

func getImageStringValue(index int, field string) string {
	return fmt.Sprintf("image_%04d_%s", index, field)
}

func getImageBasename(index int) string {
	return getImageStringValue(index, pathField)
}

func makeImageFile(i int) *file.ImageFile {
	return &file.ImageFile{
		BaseFile: &file.BaseFile{
			Path:           getFilePath(folderIdxWithImageFiles, getImageBasename(i)),
			Basename:       getImageBasename(i),
			ParentFolderID: folderIDs[folderIdxWithImageFiles],
			Fingerprints: []file.Fingerprint{
				{
					Type:        file.FingerprintTypeMD5,
					Fingerprint: getImageStringValue(i, checksumField),
				},
			},
		},
		Height: getHeight(i),
		Width:  getWidth(i),
	}
}

func makeImage(i int) *models.Image {
	title := getImageStringValue(i, titleField)
	var studioID *int
	if _, ok := imageStudios[i]; ok {
		v := studioIDs[imageStudios[i]]
		studioID = &v
	}

	gids := indexesToIDs(galleryIDs, imageGalleries[i])
	pids := indexesToIDs(performerIDs, imagePerformers[i])
	tids := indexesToIDs(tagIDs, imageTags[i])

	return &models.Image{
		Title:        title,
		Rating:       getIntPtr(getRating(i)),
		OCounter:     getOCounter(i),
		StudioID:     studioID,
		GalleryIDs:   models.NewRelatedIDs(gids),
		PerformerIDs: models.NewRelatedIDs(pids),
		TagIDs:       models.NewRelatedIDs(tids),
	}
}

func createImages(ctx context.Context, n int) error {
	qb := db.TxnRepository().Image
	fqb := db.File

	for i := 0; i < n; i++ {
		f := makeImageFile(i)
		if i == imageIdxInZip {
			f.ZipFileID = &fileIDs[fileIdxZip]
		}

		if err := fqb.Create(ctx, f); err != nil {
			return fmt.Errorf("creating image file: %w", err)
		}
		imageFileIDs = append(imageFileIDs, f.ID)

		image := makeImage(i)

		err := qb.Create(ctx, &models.ImageCreateInput{
			Image:   image,
			FileIDs: []file.ID{f.ID},
		})

		if err != nil {
			return fmt.Errorf("Error creating image %v+: %s", image, err.Error())
		}

		imageIDs = append(imageIDs, image.ID)
	}

	return nil
}

func getGalleryStringValue(index int, field string) string {
	return getPrefixedStringValue("gallery", index, field)
}

func getGalleryNullStringValue(index int, field string) sql.NullString {
	return getPrefixedNullStringValue("gallery", index, field)
}

func getGalleryNullStringPtr(index int, field string) *string {
	return getStringPtr(getPrefixedStringValue("gallery", index, field))
}

func getGalleryBasename(index int) string {
	return getGalleryStringValue(index, pathField)
}

func makeGalleryFile(i int) *file.BaseFile {
	return &file.BaseFile{
		Path:           getFilePath(folderIdxWithGalleryFiles, getGalleryBasename(i)),
		Basename:       getGalleryBasename(i),
		ParentFolderID: folderIDs[folderIdxWithGalleryFiles],
		Fingerprints: []file.Fingerprint{
			{
				Type:        file.FingerprintTypeMD5,
				Fingerprint: getGalleryStringValue(i, checksumField),
			},
		},
	}
}

func makeGallery(i int, includeScenes bool) *models.Gallery {
	var studioID *int
	if _, ok := galleryStudios[i]; ok {
		v := studioIDs[galleryStudios[i]]
		studioID = &v
	}

	pids := indexesToIDs(performerIDs, galleryPerformers[i])
	tids := indexesToIDs(tagIDs, galleryTags[i])

	ret := &models.Gallery{
		Title:        getGalleryStringValue(i, titleField),
		URL:          getGalleryNullStringValue(i, urlField).String,
		Rating:       getIntPtr(getRating(i)),
		Date:         getObjectDateObject(i),
		StudioID:     studioID,
		PerformerIDs: models.NewRelatedIDs(pids),
		TagIDs:       models.NewRelatedIDs(tids),
	}

	if includeScenes {
		ret.SceneIDs = models.NewRelatedIDs(indexesToIDs(sceneIDs, sceneGalleries.reverseLookup(i)))
	}

	return ret
}

func createGalleries(ctx context.Context, n int) error {
	gqb := db.TxnRepository().Gallery
	fqb := db.File

	for i := 0; i < n; i++ {
		var fileIDs []file.ID
		if i != galleryIdxWithoutFile {
			f := makeGalleryFile(i)
			if err := fqb.Create(ctx, f); err != nil {
				return fmt.Errorf("creating gallery file: %w", err)
			}
			galleryFileIDs = append(galleryFileIDs, f.ID)
			fileIDs = []file.ID{f.ID}
		} else {
			galleryFileIDs = append(galleryFileIDs, 0)
		}

		// gallery relationship will be created with galleries
		const includeScenes = false
		gallery := makeGallery(i, includeScenes)

		err := gqb.Create(ctx, gallery, fileIDs)

		if err != nil {
			return fmt.Errorf("Error creating gallery %v+: %s", gallery, err.Error())
		}

		galleryIDs = append(galleryIDs, gallery.ID)
	}

	return nil
}

func getMovieStringValue(index int, field string) string {
	return getPrefixedStringValue("movie", index, field)
}

func getMovieNullStringValue(index int, field string) sql.NullString {
	return getPrefixedNullStringValue("movie", index, field)
}

// createMoviees creates n movies with plain Name and o movies with camel cased NaMe included
func createMovies(ctx context.Context, mqb models.MovieReaderWriter, n int, o int) error {
	const namePlain = "Name"
	const nameNoCase = "NaMe"

	for i := 0; i < n+o; i++ {
		index := i
		name := namePlain

		if i >= n { // i<n tags get normal names
			name = nameNoCase       // i>=n movies get dup names if case is not checked
			index = n + o - (i + 1) // for the name to be the same the number (index) must be the same also
		} // so count backwards to 0 as needed
		// movies [ i ] and [ n + o - i - 1  ] should have similar names with only the Name!=NaMe part different

		name = getMovieStringValue(index, name)
		movie := models.Movie{
			Name:     sql.NullString{String: name, Valid: true},
			URL:      getMovieNullStringValue(index, urlField),
			Checksum: md5.FromString(name),
		}

		created, err := mqb.Create(ctx, movie)

		if err != nil {
			return fmt.Errorf("Error creating movie [%d] %v+: %s", i, movie, err.Error())
		}

		movieIDs = append(movieIDs, created.ID)
		movieNames = append(movieNames, created.Name.String)
	}

	return nil
}

func getPerformerStringValue(index int, field string) string {
	return getPrefixedStringValue("performer", index, field)
}

func getPerformerNullStringValue(index int, field string) string {
	ret := getPrefixedNullStringValue("performer", index, field)

	return ret.String
}

func getPerformerBoolValue(index int) bool {
	index = index % 2
	return index == 1
}

func getPerformerBirthdate(index int) *models.Date {
	const minAge = 18
	birthdate := time.Now()
	birthdate = birthdate.AddDate(-minAge-index, -1, -1)

	ret := models.Date{
		Time: birthdate,
	}
	return &ret
}

func getPerformerDeathDate(index int) *models.Date {
	if index != 5 {
		return nil
	}

	deathDate := time.Now()
	deathDate = deathDate.AddDate(-index+1, -1, -1)

	ret := models.Date{
		Time: deathDate,
	}
	return &ret
}

func getPerformerCareerLength(index int) *string {
	if index%5 == 0 {
		return nil
	}

	ret := fmt.Sprintf("20%2d", index)
	return &ret
}

func getIgnoreAutoTag(index int) bool {
	return index%5 == 0
}

func performerStashID(i int) models.StashID {
	return models.StashID{
		StashID:  getPerformerStringValue(i, "stashid"),
		Endpoint: getPerformerStringValue(i, "endpoint"),
	}
}

// createPerformers creates n performers with plain Name and o performers with camel cased NaMe included
func createPerformers(ctx context.Context, n int, o int) error {
	pqb := db.Performer

	const namePlain = "Name"
	const nameNoCase = "NaMe"

	name := namePlain

	for i := 0; i < n+o; i++ {
		index := i

		if i >= n { // i<n tags get normal names
			name = nameNoCase       // i>=n performers get dup names if case is not checked
			index = n + o - (i + 1) // for the name to be the same the number (index) must be the same also
		} // so count backwards to 0 as needed
		// performers [ i ] and [ n + o - i - 1  ] should have similar names with only the Name!=NaMe part different

		performer := models.Performer{
			Name:          getPerformerStringValue(index, name),
			Checksum:      getPerformerStringValue(i, checksumField),
			URL:           getPerformerNullStringValue(i, urlField),
			Favorite:      getPerformerBoolValue(i),
			Birthdate:     getPerformerBirthdate(i),
			DeathDate:     getPerformerDeathDate(i),
			Details:       getPerformerStringValue(i, "Details"),
			Ethnicity:     getPerformerStringValue(i, "Ethnicity"),
			Rating:        getIntPtr(getRating(i)),
			IgnoreAutoTag: getIgnoreAutoTag(i),
		}

		careerLength := getPerformerCareerLength(i)
		if careerLength != nil {
			performer.CareerLength = *careerLength
		}

		err := pqb.Create(ctx, &performer)

		if err != nil {
			return fmt.Errorf("Error creating performer %v+: %s", performer, err.Error())
		}

		if (index+1)%5 != 0 {
			if err := pqb.UpdateStashIDs(ctx, performer.ID, []models.StashID{
				performerStashID(i),
			}); err != nil {
				return fmt.Errorf("setting performer stash ids: %w", err)
			}
		}

		performerIDs = append(performerIDs, performer.ID)
		performerNames = append(performerNames, performer.Name)
	}

	return nil
}

func getTagStringValue(index int, field string) string {
	return "tag_" + strconv.FormatInt(int64(index), 10) + "_" + field
}

func getTagSceneCount(id int) int {
	if id == tagIDs[tagIdx1WithScene] || id == tagIDs[tagIdx2WithScene] || id == tagIDs[tagIdxWithScene] || id == tagIDs[tagIdx3WithScene] {
		return 1
	}

	return 0
}

func getTagMarkerCount(id int) int {
	if id == tagIDs[tagIdxWithPrimaryMarkers] {
		return 3
	}

	if id == tagIDs[tagIdxWithMarkers] {
		return 1
	}

	return 0
}

func getTagImageCount(id int) int {
	if id == tagIDs[tagIdx1WithImage] || id == tagIDs[tagIdx2WithImage] || id == tagIDs[tagIdxWithImage] {
		return 1
	}

	return 0
}

func getTagGalleryCount(id int) int {
	if id == tagIDs[tagIdx1WithGallery] || id == tagIDs[tagIdx2WithGallery] || id == tagIDs[tagIdxWithGallery] {
		return 1
	}

	return 0
}

func getTagPerformerCount(id int) int {
	if id == tagIDs[tagIdx1WithPerformer] || id == tagIDs[tagIdx2WithPerformer] || id == tagIDs[tagIdxWithPerformer] {
		return 1
	}

	return 0
}

func getTagParentCount(id int) int {
	if id == tagIDs[tagIdxWithParentTag] || id == tagIDs[tagIdxWithGrandParent] || id == tagIDs[tagIdxWithParentAndChild] {
		return 1
	}

	return 0
}

func getTagChildCount(id int) int {
	if id == tagIDs[tagIdxWithChildTag] || id == tagIDs[tagIdxWithGrandChild] || id == tagIDs[tagIdxWithParentAndChild] {
		return 1
	}

	return 0
}

// createTags creates n tags with plain Name and o tags with camel cased NaMe included
func createTags(ctx context.Context, tqb models.TagReaderWriter, n int, o int) error {
	const namePlain = "Name"
	const nameNoCase = "NaMe"

	name := namePlain

	for i := 0; i < n+o; i++ {
		index := i

		if i >= n { // i<n tags get normal names
			name = nameNoCase       // i>=n tags get dup names if case is not checked
			index = n + o - (i + 1) // for the name to be the same the number (index) must be the same also
		} // so count backwards to 0 as needed
		// tags [ i ] and [ n + o - i - 1  ] should have similar names with only the Name!=NaMe part different

		tag := models.Tag{
			Name:          getTagStringValue(index, name),
			IgnoreAutoTag: getIgnoreAutoTag(i),
		}

		created, err := tqb.Create(ctx, tag)

		if err != nil {
			return fmt.Errorf("Error creating tag %v+: %s", tag, err.Error())
		}

		// add alias
		alias := getTagStringValue(i, "Alias")
		if err := tqb.UpdateAliases(ctx, created.ID, []string{alias}); err != nil {
			return fmt.Errorf("error setting tag alias: %s", err.Error())
		}

		tagIDs = append(tagIDs, created.ID)
		tagNames = append(tagNames, created.Name)
	}

	return nil
}

func getStudioStringValue(index int, field string) string {
	return getPrefixedStringValue("studio", index, field)
}

func getStudioNullStringValue(index int, field string) sql.NullString {
	return getPrefixedNullStringValue("studio", index, field)
}

func createStudio(ctx context.Context, sqb models.StudioReaderWriter, name string, parentID *int64) (*models.Studio, error) {
	studio := models.Studio{
		Name:     sql.NullString{String: name, Valid: true},
		Checksum: md5.FromString(name),
	}

	if parentID != nil {
		studio.ParentID = sql.NullInt64{Int64: *parentID, Valid: true}
	}

	return createStudioFromModel(ctx, sqb, studio)
}

func createStudioFromModel(ctx context.Context, sqb models.StudioReaderWriter, studio models.Studio) (*models.Studio, error) {
	created, err := sqb.Create(ctx, studio)

	if err != nil {
		return nil, fmt.Errorf("Error creating studio %v+: %s", studio, err.Error())
	}

	return created, nil
}

// createStudios creates n studios with plain Name and o studios with camel cased NaMe included
func createStudios(ctx context.Context, sqb models.StudioReaderWriter, n int, o int) error {
	const namePlain = "Name"
	const nameNoCase = "NaMe"

	for i := 0; i < n+o; i++ {
		index := i
		name := namePlain

		if i >= n { // i<n studios get normal names
			name = nameNoCase       // i>=n studios get dup names if case is not checked
			index = n + o - (i + 1) // for the name to be the same the number (index) must be the same also
		} // so count backwards to 0 as needed
		// studios [ i ] and [ n + o - i - 1  ] should have similar names with only the Name!=NaMe part different

		name = getStudioStringValue(index, name)
		studio := models.Studio{
			Name:          sql.NullString{String: name, Valid: true},
			Checksum:      md5.FromString(name),
			URL:           getStudioNullStringValue(index, urlField),
			IgnoreAutoTag: getIgnoreAutoTag(i),
		}
		created, err := createStudioFromModel(ctx, sqb, studio)

		if err != nil {
			return err
		}

		// add alias
		// only add aliases for some scenes
		if i == studioIdxWithMovie || i%5 == 0 {
			alias := getStudioStringValue(i, "Alias")
			if err := sqb.UpdateAliases(ctx, created.ID, []string{alias}); err != nil {
				return fmt.Errorf("error setting studio alias: %s", err.Error())
			}
		}

		studioIDs = append(studioIDs, created.ID)
		studioNames = append(studioNames, created.Name.String)
	}

	return nil
}

func createMarker(ctx context.Context, mqb models.SceneMarkerReaderWriter, markerSpec markerSpec) error {
	marker := models.SceneMarker{
		SceneID:      sql.NullInt64{Int64: int64(sceneIDs[markerSpec.sceneIdx]), Valid: true},
		PrimaryTagID: tagIDs[markerSpec.primaryTagIdx],
	}

	created, err := mqb.Create(ctx, marker)

	if err != nil {
		return fmt.Errorf("error creating marker %v+: %w", marker, err)
	}

	markerIDs = append(markerIDs, created.ID)

	if len(markerSpec.tagIdxs) > 0 {
		newTagIDs := []int{}

		for _, tagIdx := range markerSpec.tagIdxs {
			newTagIDs = append(newTagIDs, tagIDs[tagIdx])
		}

		if err := mqb.UpdateTags(ctx, created.ID, newTagIDs); err != nil {
			return fmt.Errorf("error creating marker/tag join: %w", err)
		}
	}

	return nil
}

func getSavedFilterMode(index int) models.FilterMode {
	switch index {
	case savedFilterIdxScene, savedFilterIdxDefaultScene:
		return models.FilterModeScenes
	case savedFilterIdxImage, savedFilterIdxDefaultImage:
		return models.FilterModeImages
	default:
		return models.FilterModeScenes
	}
}

func getSavedFilterName(index int) string {
	if index <= savedFilterIdxDefaultImage {
		// empty string for default filters
		return ""
	}

	if index <= savedFilterIdxImage {
		// use the same name for the first two - should be possible
		return firstSavedFilterName
	}

	return getPrefixedStringValue("savedFilter", index, "Name")
}

func createSavedFilters(ctx context.Context, qb models.SavedFilterReaderWriter, n int) error {
	for i := 0; i < n; i++ {
		savedFilter := models.SavedFilter{
			Mode:   getSavedFilterMode(i),
			Name:   getSavedFilterName(i),
			Filter: getPrefixedStringValue("savedFilter", i, "Filter"),
		}

		created, err := qb.Create(ctx, savedFilter)

		if err != nil {
			return fmt.Errorf("Error creating saved filter %v+: %s", savedFilter, err.Error())
		}

		savedFilterIDs = append(savedFilterIDs, created.ID)
	}

	return nil
}

func doLinks(links [][2]int, fn func(idx1, idx2 int) error) error {
	for _, l := range links {
		if err := fn(l[0], l[1]); err != nil {
			return err
		}
	}

	return nil
}

func linkPerformerTags(ctx context.Context) error {
	qb := db.Performer
	return doLinks(performerTagLinks, func(performerIndex, tagIndex int) error {
		performerID := performerIDs[performerIndex]
		tagID := tagIDs[tagIndex]
		tagIDs, err := qb.GetTagIDs(ctx, performerID)
		if err != nil {
			return err
		}

		tagIDs = intslice.IntAppendUnique(tagIDs, tagID)

		return qb.UpdateTags(ctx, performerID, tagIDs)
	})
}

func linkMovieStudios(ctx context.Context, mqb models.MovieWriter) error {
	return doLinks(movieStudioLinks, func(movieIndex, studioIndex int) error {
		movie := models.MoviePartial{
			ID:       movieIDs[movieIndex],
			StudioID: &sql.NullInt64{Int64: int64(studioIDs[studioIndex]), Valid: true},
		}
		_, err := mqb.Update(ctx, movie)

		return err
	})
}

func linkStudiosParent(ctx context.Context, qb models.StudioWriter) error {
	return doLinks(studioParentLinks, func(parentIndex, childIndex int) error {
		studio := models.StudioPartial{
			ID:       studioIDs[childIndex],
			ParentID: &sql.NullInt64{Int64: int64(studioIDs[parentIndex]), Valid: true},
		}
		_, err := qb.Update(ctx, studio)

		return err
	})
}

func linkTagsParent(ctx context.Context, qb models.TagReaderWriter) error {
	return doLinks(tagParentLinks, func(parentIndex, childIndex int) error {
		tagID := tagIDs[childIndex]
		parentTags, err := qb.FindByChildTagID(ctx, tagID)
		if err != nil {
			return err
		}

		var parentIDs []int
		for _, parentTag := range parentTags {
			parentIDs = append(parentIDs, parentTag.ID)
		}

		parentIDs = append(parentIDs, tagIDs[parentIndex])

		return qb.UpdateParentTags(ctx, tagID, parentIDs)
	})
}

func addTagImage(ctx context.Context, qb models.TagWriter, tagIndex int) error {
	return qb.UpdateImage(ctx, tagIDs[tagIndex], models.DefaultTagImage)
}
