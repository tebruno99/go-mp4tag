package mp4tag

import (
	"encoding/binary"
	"fmt"
	"io"
	"strconv"
	"strings"
	"time"
)

func (boxes MP4Boxes) getBoxByPath(boxPath string) *MP4Box {
	for _, box := range boxes.Boxes {
		if box.Path == boxPath {
			return box
		}
	}
	return nil
}

func (boxes MP4Boxes) getBoxesByPath(boxPath string) []*MP4Box {
	var outBoxes []*MP4Box
	for _, box := range boxes.Boxes {
		if box.Path == boxPath {
			outBoxes = append(outBoxes, box)
		}
	}
	return outBoxes
}

func (mp4 MP4) readString(size int64) (string, error) {
	buf := make([]byte, size)
	_, err := io.ReadFull(mp4.f, buf)
	if err != nil {
		return "", err
	}
	return string(buf), nil
}

func (mp4 MP4) readBoxName() (string, error) {
	buf := make([]byte, 4)
	_, err := io.ReadFull(mp4.f, buf)
	if err != nil {
		return "", err
	}
	boxName := string(buf)
	if buf[0] == 0xA9 {
		boxName = "(c)" + strings.ToLower(boxName[1:])
	}
	return boxName, nil
}

func (mp4 MP4) readI16BE() (int16, error) {
	buf := make([]byte, 2)
	_, err := io.ReadFull(mp4.f, buf)
	if err != nil {
		return -1, err
	}
	num := binary.BigEndian.Uint16(buf)
	return int16(num), nil
}

func (mp4 MP4) readI32BE() (int32, error) {
	buf := make([]byte, 4)
	_, err := io.ReadFull(mp4.f, buf)
	if err != nil {
		return -1, err
	}
	num := binary.BigEndian.Uint32(buf)
	return int32(num), nil
}

func (mp4 MP4) readI64BE() (int64, error) {
	buf := make([]byte, 8)
	_, err := io.ReadFull(mp4.f, buf)
	if err != nil {
		return -1, err
	}
	num := binary.BigEndian.Uint64(buf)
	return int64(num), nil
}

func (mp4 MP4) readBoxes(boxes MP4Boxes, parentEndsAt, level int64, p string) (MP4Boxes, error) {
	empty := MP4Boxes{}
	pos, err := getPos(mp4.f)
	if err != nil {
		return empty, err
	}
	if pos >= parentEndsAt {
		return boxes, err
	}
	boxSizeI32, err := mp4.readI32BE()
	if err != nil {
		return empty, err
	}
	boxName, err := mp4.readBoxName()
	if err != nil {
		return empty, err
	}
	boxSize := int64(boxSizeI32)
	endsAt := pos + boxSize
	if boxName == "meta" {
		_, err = mp4.f.Seek(4, io.SeekCurrent)
		if err != nil {
			return empty, err
		}
	}
	p += "." + boxName
	box := &MP4Box{
		StartOffset: pos,
		EndOffset:   endsAt,
		BoxSize:     boxSize,
		Path:        p[1:],
	}
	boxes.Boxes = append(boxes.Boxes, box)
	if containsStr(containers, boxName) {
		boxes, err = mp4.readBoxes(boxes, endsAt, level+1, p)
		if err != nil {
			return empty, err
		}
	}
	p = p[:len(p)-len(boxName)-1]
	_, err = mp4.f.Seek(pos+boxSize, io.SeekStart)
	if err != nil {
		return empty, err
	}
	boxes, err = mp4.readBoxes(boxes, parentEndsAt, level, p)
	return boxes, err
}

func checkBoxes(boxes MP4Boxes) error {
	paths := [5]string{
		"moov", "mdat", "moov.udta", "moov.udta.meta",
		"moov.trak.mdia.minf.stbl.stco",
	}
	for _, path := range paths {
		if boxes.getBoxByPath(path) == nil {
			return &ErrBoxNotPresent{Msg: path + " box not present"}
		}
	}
	return nil
}

func (mp4 MP4) readDuration(boxes MP4Boxes) (time.Duration, error) {
	mvhd := boxes.getBoxByPath("moov.mvhd")
	mp4.f.Seek(mvhd.StartOffset+20, io.SeekStart)
	timeScale, err := mp4.readI32BE()
	if err != nil {
		return 0, err
	}
	timeUnit, err := mp4.readI32BE()
	if err != nil {
		return 0, err
	}
	microseconds := (float64(timeUnit) / float64(timeScale)) * 1000000
	var d time.Duration = time.Duration(microseconds * float64(time.Microsecond))
	return d, nil
}

type SamplesPerChunk struct {
	firstChunk int
	numSamples int
}

func (mp4 MP4) readNeroChap(boxes MP4Boxes) ([]Chapter, error) {
	chapters := []Chapter{}
	chplBox := boxes.getBoxByPath("moov.udta.chpl")

	// Skip version flags and reserved bytes
	_, err := mp4.f.Seek(chplBox.StartOffset+13, io.SeekStart)
	if err != nil {
		return chapters, err
	}

	chapterCount, err := mp4.readI32BE()
	if err != nil {
		return chapters, err
	}

	for i := 0; i < int(chapterCount); i++ {
		chapterStartTime, err := mp4.readI64BE()
		if err != nil {
			return chapters, err
		}
		b, err := mp4.readByte()
		if err != nil {
			return chapters, err
		}
		titleLength := uint8(b)

		title, err := mp4.readString(int64(titleLength))
		if err != nil {
			return chapters, err
		}
		// 100 nanosecond base for some reason
		chapter := Chapter{StartTime: time.Duration(chapterStartTime * 100), Title: title}
		if len(chapters) > 0 {
			lastChapter := &chapters[len(chapters)-1]
			lastChapter.Duration = chapter.StartTime - lastChapter.StartTime

		}
		chapters = append(chapters, chapter)
	}

	return chapters, nil
}

func (mp4 MP4) readChap(boxes MP4Boxes) ([]Chapter, error) {
	// Read nero chapters rather than Quicktime if they exist
	if boxes.getBoxByPath("moov.udta.chpl") != nil {
		return mp4.readNeroChap(boxes)
	}

	chapters := []Chapter{}
	hdlrBoxes := boxes.getBoxesByPath("moov.trak.mdia.hdlr")

	// Have to make sure the the trak is subType "text"
	mdiaIndex := -1
	for i, box := range hdlrBoxes {
		mp4.f.Seek(box.StartOffset+16, io.SeekStart)
		subType, err := mp4.readString(4)
		if err != nil {
			return chapters, err
		}
		if subType == "text" {
			mdiaIndex = i
		}
	}

	// No chapter trak
	if mdiaIndex == -1 {
		return chapters, nil
	}

	var mdhdBox *MP4Box
	var stscBox *MP4Box
	var stcoBox *MP4Box
	var sttsBox *MP4Box

	var samplesPerChunk []SamplesPerChunk

	mdhdBoxes := boxes.getBoxesByPath("moov.trak.mdia.mdhd")
	for i, box := range mdhdBoxes {
		if i == mdiaIndex {
			mdhdBox = box
		}
	}
	if mdhdBox == nil {
		return chapters, &ErrInvalidChapter{}
	}

	mp4.f.Seek(mdhdBox.StartOffset+20, io.SeekStart)
	timeScale, err := mp4.readI32BE()
	if err != nil {
		return chapters, err
	}

	stscBoxes := boxes.getBoxesByPath("moov.trak.mdia.minf.stbl.stsc")
	for i, box := range stscBoxes {
		if i == mdiaIndex {
			stscBox = box
		}
	}
	if stscBox == nil {
		return chapters, &ErrInvalidChapter{}
	}

	stcoBoxes := boxes.getBoxesByPath("moov.trak.mdia.minf.stbl.stco")
	for i, box := range stcoBoxes {
		if i == mdiaIndex {
			stcoBox = box
		}
	}
	if stcoBox == nil {
		return chapters, &ErrInvalidChapter{}
	}

	sttsBoxes := boxes.getBoxesByPath("moov.trak.mdia.minf.stbl.stts")
	for i, box := range sttsBoxes {
		if i == mdiaIndex {
			sttsBox = box
		}
	}
	if sttsBox == nil {
		return chapters, &ErrInvalidChapter{}
	}

	_, err = mp4.f.Seek(stscBox.StartOffset+12, io.SeekStart)
	if err != nil {
		return chapters, err
	}

	numEntries, err := mp4.readI32BE()
	if err != nil {
		return chapters, err
	}

	for i := 0; i < int(numEntries); i++ {
		firstChunk, err := mp4.readI32BE()
		if err != nil {
			return chapters, err
		}
		numSamples, err := mp4.readI32BE()
		if err != nil {
			return chapters, err
		}
		mp4.f.Seek(4, io.SeekCurrent)
		samplesPerChunk = append(samplesPerChunk, SamplesPerChunk{firstChunk: int(firstChunk), numSamples: int(numSamples)})
	}

	_, err = mp4.f.Seek(stcoBox.StartOffset+12, io.SeekStart)
	if err != nil {
		return chapters, err
	}
	numEntries, err = mp4.readI32BE()
	if err != nil {
		return chapters, err
	}

	chunkOffsets := []int{}
	for i := 0; i < int(numEntries); i++ {
		chunkOffset, err := mp4.readI32BE()
		if err != nil {
			return chapters, err
		}
		chunkOffsets = append(chunkOffsets, int(chunkOffset))
	}

	chapterTitles := []string{}
	for i, offset := range chunkOffsets {
		var chunkNumber = i + 1

		var numSamples = 1

		for _, record := range samplesPerChunk {
			if record.firstChunk > chunkNumber {
				break
			}
			numSamples = record.numSamples
		}

		mp4.f.Seek(int64(offset), io.SeekStart)

		for j := 0; j < numSamples; j++ {
			length, err := mp4.readI16BE()
			if err != nil {
				return chapters, err
			}

			if length <= 0 {
				return chapters, err
			}
			chapterTitle, err := mp4.readString(int64(length))

			if err != nil {
				return chapters, err
			}
			chapterTitles = append(chapterTitles, chapterTitle)
		}
	}

	_, err = mp4.f.Seek(sttsBox.StartOffset+12, io.SeekStart)
	if err != nil {
		return chapters, err
	}
	numEntries, err = mp4.readI32BE()
	if err != nil {
		return chapters, err
	}
	if int(numEntries) != len(chapterTitles) {
		return chapters, &ErrInvalidChapter{}
	}

	var start time.Duration = 0
	for i := 0; i < int(numEntries); i++ {
		mp4.f.Seek(4, io.SeekCurrent)
		durationMs, err := mp4.readI32BE()
		if err != nil {
			return chapters, err
		}
		var duration = time.Duration(float64(durationMs) / float64(timeScale) * float64(time.Second))
		chapter := Chapter{
			Title:     chapterTitles[i],
			StartTime: start,
			Duration:  duration,
		}
		chapters = append(chapters, chapter)
		start += duration
	}

	return chapters, nil
}

func (mp4 MP4) readTag(boxes MP4Boxes, boxName string) (string, error) {
	path := fmt.Sprintf("moov.udta.meta.ilst.%s.data", boxName)
	box := boxes.getBoxByPath(path)
	if box == nil {
		return "", nil
	}
	_, err := mp4.f.Seek(box.StartOffset+16, io.SeekStart)
	if err != nil {
		return "", err
	}
	tag, err := mp4.readString(box.BoxSize - 16)
	return tag, err
}

func (mp4 MP4) readByte() (byte, error) {
	buf := make([]byte, 1)
	_, err := mp4.f.Read(buf)
	if err != nil {
		return 0x0, err
	}
	return buf[0], nil
}

func (mp4 MP4) readBPM(boxes MP4Boxes) (int16, error) {
	box := boxes.getBoxByPath("moov.udta.meta.ilst.tmpo.data")
	if box == nil {
		return -1, nil
	}
	_, err := mp4.f.Seek(box.StartOffset+16, io.SeekStart)
	if err != nil {
		return -1, err
	}
	bpm, err := mp4.readI16BE()
	return bpm, err
}

func (mp4 MP4) readPics(_boxes MP4Boxes) ([]*MP4Picture, error) {
	var outPics []*MP4Picture
	boxes := _boxes.getBoxesByPath("moov.udta.meta.ilst.covr.data")
	if boxes == nil {
		return nil, nil
	}
	for _, box := range boxes {
		var pic MP4Picture
		_, err := mp4.f.Seek(box.StartOffset+11, io.SeekStart)
		if err != nil {
			return nil, err
		}
		b, err := mp4.readByte()
		if err != nil {
			return nil, err
		}

		imageType, ok := resolveImageType[uint8(b)]
		if ok {
			if imageType == ImageTypeJPEG {
				pic.Format = ImageTypeJPEG
			} else {
				pic.Format = ImageTypePNG
			}
		}
		_, err = mp4.f.Seek(4, io.SeekCurrent)
		if err != nil {
			return nil, err
		}
		buf := make([]byte, box.BoxSize-16)
		_, err = io.ReadFull(mp4.f, buf)
		if err != nil {
			return nil, err
		}
		pic.Data = buf
		outPics = append(outPics, &pic)
	}
	return outPics, nil
}

func (mp4 MP4) readTrknDisk(boxes MP4Boxes, boxName string) (int16, int16, error) {
	path := fmt.Sprintf("moov.udta.meta.ilst.%s.data", boxName)
	box := boxes.getBoxByPath(path)
	if box == nil {
		return -1, -1, nil
	}
	_, err := mp4.f.Seek(box.StartOffset+18, io.SeekStart)
	if err != nil {
		return -1, -1, nil
	}

	num, err := mp4.readI16BE()
	if err != nil {
		return -1, -1, nil
	}
	total, err := mp4.readI16BE()
	if err != nil {
		return -1, -1, nil
	}
	return num, total, nil
}

func addToOthers(others map[string][]string, key, val string) map[string][]string {
	existingOthers, ok := others[key]
	if ok {
		existingOthers = append(existingOthers, val)
		others[key] = existingOthers
	} else {
		others[key] = []string{val}
	}
	return others
}

func (mp4 MP4) readCustom(boxes MP4Boxes) (map[string]string, map[string][]string, error) {
	var (
		names  []string
		values []string
	)
	path := "moov.udta.meta.ilst.----"
	nameBoxes := boxes.getBoxesByPath(path + ".name")
	if nameBoxes == nil {
		return nil, nil, nil
	}
	for _, box := range nameBoxes {
		_, err := mp4.f.Seek(box.StartOffset+12, io.SeekStart)
		if err != nil {
			return nil, nil, err
		}
		name, err := mp4.readString(box.BoxSize - 12)
		if err != nil {
			return nil, nil, err
		}
		if mp4.upperCustom {
			name = strings.ToUpper(name)
		}
		names = append(names, name)
	}

	others := map[string][]string{}

	dataBoxes := boxes.getBoxesByPath(path + ".data")

	var (
		prev int64
		idx  int
	)

	for _, box := range dataBoxes {
		_, err := mp4.f.Seek(box.StartOffset+16, io.SeekStart)
		if err != nil {
			return nil, nil, err
		}
		value, err := mp4.readString(box.BoxSize - 16)
		if err != nil {
			return nil, nil, err
		}
		if box.StartOffset == prev {
			others = addToOthers(others, names[idx-1], value)
			prev = box.EndOffset
			continue
		}
		values = append(values, value)
		prev = box.EndOffset
		idx++
	}

	custom := map[string]string{}
	for idx, name := range names {
		_, ok := custom[name]
		if ok {
			existingOthers, ok := others[name]
			if ok {
				existingOthers = append(existingOthers, values[idx])
				others[name] = existingOthers
			} else {
				others[name] = []string{values[idx]}
			}
		} else {
			custom[name] = values[idx]
		}
	}
	return custom, others, nil
}

func (mp4 MP4) readITAlbumID(boxes MP4Boxes) (int32, error) {
	box := boxes.getBoxByPath("moov.udta.meta.ilst.plID.data")
	if box == nil {
		return -1, nil
	}
	_, err := mp4.f.Seek(box.StartOffset+20, io.SeekStart)
	if err != nil {
		return -1, err
	}
	id, err := mp4.readI32BE()
	return id, err
}

func (mp4 MP4) readITArtistID(boxes MP4Boxes) (int32, error) {
	box := boxes.getBoxByPath("moov.udta.meta.ilst.atID.data")
	if box == nil {
		return -1, nil
	}
	_, err := mp4.f.Seek(box.StartOffset+16, io.SeekStart)
	if err != nil {
		return -1, err
	}
	id, err := mp4.readI32BE()
	return id, err
}

func (mp4 MP4) readAdvisory(boxes MP4Boxes) (ItunesAdvisory, error) {
	none := ItunesAdvisoryNone
	box := boxes.getBoxByPath("moov.udta.meta.ilst.rtng.data")
	if box == nil {
		return none, nil
	}
	_, err := mp4.f.Seek(box.StartOffset+16, io.SeekStart)
	if err != nil {
		return none, err
	}
	b, err := mp4.readByte()
	if err != nil {
		return none, err
	}
	advisory, ok := resolveItunesAdvisory[uint8(b)]
	if !ok {
		return none, nil
	}
	return advisory, nil
}

func (mp4 MP4) readGenre(boxes MP4Boxes) (Genre, error) {
	none := GenreNone
	box := boxes.getBoxByPath("moov.udta.meta.ilst.gnre.data")
	if box == nil {
		return none, nil
	}
	_, err := mp4.f.Seek(box.StartOffset+17, io.SeekStart)
	if err != nil {
		return none, err
	}
	b, err := mp4.readByte()
	if err != nil {
		return none, err
	}
	genre, ok := resolveGenre[uint8(b)]
	if !ok {
		return none, nil
	}
	return genre, nil
}

func (mp4 MP4) readTags(boxes MP4Boxes) (*MP4Tags, error) {
	album, err := mp4.readTag(boxes, "(c)alb")
	if err != nil {
		return nil, err
	}
	albumArtist, err := mp4.readTag(boxes, "aART")
	if err != nil {
		return nil, err
	}
	artist, err := mp4.readTag(boxes, "(c)art")
	if err != nil {
		return nil, err
	}
	bpm, err := mp4.readBPM(boxes)
	if err != nil {
		return nil, err
	}

	chapters, err := mp4.readChap(boxes)
	if err != nil {
		return nil, err
	}

	duration, err := mp4.readDuration(boxes)
	if err != nil {
		return nil, err
	}

	comment, err := mp4.readTag(boxes, "(c)cmt")
	if err != nil {
		return nil, err
	}
	composer, err := mp4.readTag(boxes, "(c)wrt")
	if err != nil {
		return nil, err
	}
	conductor, err := mp4.readTag(boxes, "(c)con")
	if err != nil {
		return nil, err
	}
	copyright, err := mp4.readTag(boxes, "cprt")
	if err != nil {
		return nil, err
	}
	custom, otherCustom, err := mp4.readCustom(boxes)
	if err != nil {
		return nil, err
	}
	customGenre, err := mp4.readTag(boxes, "(c)gen")
	if err != nil {
		return nil, err
	}
	description, err := mp4.readTag(boxes, "desc")
	if err != nil {
		return nil, err
	}
	lyrics, err := mp4.readTag(boxes, "(c)lyr")
	if err != nil {
		return nil, err
	}
	narrator, err := mp4.readTag(boxes, "(c)nrt")
	if err != nil {
		return nil, err
	}
	publisher, err := mp4.readTag(boxes, "(c)pub")
	if err != nil {
		return nil, err
	}
	title, err := mp4.readTag(boxes, "(c)nam")
	if err != nil {
		return nil, err
	}

	pics, err := mp4.readPics(boxes)
	if err != nil {
		return nil, err
	}
	trackNum, trackTotal, err := mp4.readTrknDisk(boxes, "trkn")
	if err != nil {
		return nil, err
	}
	discNum, discTotal, err := mp4.readTrknDisk(boxes, "disk")
	if err != nil {
		return nil, err
	}

	genre, err := mp4.readGenre(boxes)
	if err != nil {
		return nil, err
	}

	advisory, err := mp4.readAdvisory(boxes)
	if err != nil {
		return nil, err
	}

	albumID, err := mp4.readITAlbumID(boxes)
	if err != nil {
		return nil, err
	}

	artistID, err := mp4.readITArtistID(boxes)
	if err != nil {
		return nil, err
	}

	tags := &MP4Tags{
		Album:          album,
		AlbumArtist:    albumArtist,
		Artist:         artist,
		BPM:            bpm,
		Comment:        comment,
		Composer:       composer,
		Conductor:      conductor,
		Copyright:      copyright,
		Custom:         custom,
		CustomGenre:    customGenre,
		Description:    description,
		DiscNumber:     discNum,
		DiscTotal:      discTotal,
		Genre:          genre,
		ItunesAdvisory: advisory,
		ItunesAlbumID:  albumID,
		ItunesArtistID: artistID,
		Lyrics:         lyrics,
		Narrator:       narrator,
		OtherCustom:    otherCustom,
		Pictures:       pics,
		Publisher:      publisher,
		Title:          title,
		TrackNumber:    trackNum,
		TrackTotal:     trackTotal,
		Chapters:       chapters,
		Duration:       duration,
	}

	year, err := mp4.readTag(boxes, "(c)day")
	if err != nil {
		return nil, err
	}

	if year != "" {
		if containsOnlyNums(year) {
			yearInt, err := strconv.ParseInt(year, 10, 32)
			if err != nil {
				return nil, err
			}
			tags.Year = int32(yearInt)
		} else {
			tags.Date = year
		}
	}

	return tags, nil
}

func (mp4 MP4) actualRead() (*MP4Tags, MP4Boxes, error) {
	var boxes MP4Boxes
	_, err := mp4.f.Seek(0, io.SeekStart)
	if err != nil {
		return nil, boxes, err
	}
	boxes, err = mp4.readBoxes(boxes, mp4.size, 0, "")
	if err != nil {
		return nil, boxes, err
	}
	err = checkBoxes(boxes)
	if err != nil {
		return nil, boxes, err
	}
	if boxes.getBoxByPath("moov.udta.meta.ilst") == nil {
		return &MP4Tags{}, boxes, nil
	}
	tags, err := mp4.readTags(boxes)
	return tags, boxes, err
}
