package mp4tag

import "os"

type ErrBoxNotPresent struct {
    Msg  string
}

type ErrUnsupportedFtyp struct {
    Msg  string
}

type ErrInvalidStcoSize struct {}

type ErrInvalidMagic struct {}


func (e *ErrBoxNotPresent) Error() string { 
    return e.Msg
}

func (e *ErrUnsupportedFtyp) Error() string { 
    return e.Msg
}

func (_ *ErrInvalidStcoSize) Error() string {
	return "stco size is invalid"
}

func (_ *ErrInvalidMagic) Error() string {
	return "file header is corrupted or not an mp4 file"
}

var ftyps = [8][]byte{
	{0x4D, 0x34, 0x41, 0x20}, // M4A
	{0x4D, 0x34, 0x42, 0x20}, // M4B
	{0x64, 0x61, 0x73, 0x68}, // dash
	{0x6D, 0x70, 0x34, 0x31}, // mp41
	{0x6D, 0x70, 0x34, 0x32}, // mp42
	{0x69, 0x73, 0x6F, 0x6D}, // isom
	{0x69, 0x73, 0x6F, 0x32}, // iso2
	{0x61, 0x76, 0x63, 0x31}, // avc1
}

var containers = []string{
  "moov", "udta", "meta", "ilst", "----", "(c)alb",
  "aART", "(c)art", "(c)nam", "(c)cmt", "(c)gen", "gnre",
  "(c)wrt", "(c)con", "cprt", "desc", "(c)lyr", "(c)nrt",
  "(c)pub", "trkn", "covr", "(c)day", "disk", "(c)too",
  "trak", "mdia", "minf", "stbl", "rtng", "plID",
  "atID", "tmpo", "sonm", "soal", "soar", "soco",
  "soaa", "tvsn", "tvsh", "tves", "tven", "tvnn", "stik",
  "ldes",
}

// 0-9
var numbers = []rune{
	0x30, 0x31, 0x32, 0x33, 0x34,
	0x35, 0x36, 0x37, 0x38, 0x39,
}

type MP4 struct {
	f *os.File
	path string
	size int64
	upperCustom bool
}

type MP4Box struct {
	StartOffset int64
	EndOffset   int64
	BoxSize     int64
	Path        string
}

type MP4Boxes struct {
	Boxes []*MP4Box
}

type ImageType int8
const (
	ImageTypeJPEG ImageType = iota + 13
	ImageTypePNG
	ImageTypeAuto
)

var resolveImageType = map[uint8]ImageType{
	13: ImageTypeJPEG,
	14: ImageTypePNG,
}

type ItunesAdvisory int8
const (
	ItunesAdvisoryNone ItunesAdvisory = iota
	ItunesAdvisoryExplicit
	ItunesAdvisoryClean
)

var resolveItunesAdvisory = map[uint8]ItunesAdvisory{
	1: ItunesAdvisoryExplicit,
	2: ItunesAdvisoryClean,
}

// iTunes stik
type ItunesStik int8
const (
	HomeVideo       ItunesStik = 0
	Normal          ItunesStik = 1
	Audiobook       ItunesStik = 2
	WhackedBookmark ItunesStik = 5
	MusicVideo      ItunesStik = 6
	Movie           ItunesStik = 9
	TvShow          ItunesStik = 10
	Booklet         ItunesStik = 11
	RingTone        ItunesStik = 14
	Podcast         ItunesStik = 21
	iTunesU         ItunesStik = 23
)

var resolveItunesStik = map[uint8]ItunesStik{
	0:  HomeVideo,
	1:  Normal,
	2:  Audiobook,
	5:  WhackedBookmark,
	6:  MusicVideo,
	9:  Movie,
	10: TvShow,
	11: Booklet,
	14: RingTone,
	21: Podcast,
	23: iTunesU,
}

var displayItunesStik = map[ItunesStik]string{
	HomeVideo:       "Home Video",
	Normal:          "Normal",
	Audiobook:       "Audiobook",
	WhackedBookmark: "Whacked Bookmark",
	MusicVideo:      "Music Video",
	Movie:           "Movie",
	TvShow:          "TV Show",
	Booklet:         "Booklet",
	RingTone:        "Ring Tone",
	Podcast:         "Podcast",
	iTunesU:         "iTunesU",
}

// GenreNone
type Genre int8
const (
	GenreNone Genre = iota
	GenreBlues
	GenreClassicRock
	GenreCountry
	GenreDance
	GenreDisco
	GenreFunk
	GenreGrunge
	GenreHipHop
	GenreJazz
	GenreMetal
	GenreNewAge
	GenreOldies
	GenreOther
	GenrePop
	GenreRhythmAndBlues
	GenreRap
	GenreReggae
	GenreRock
	GenreTechno
	GenreIndustrial
	GenreAlternative
	GenreSka
	GenreDeathMetal
	GenrePranks
	GenreSoundtrack
	GenreEurotechno
	GenreAmbient
	GenreTripHop
	GenreVocal
	GenreJassAndFunk
	GenreFusion
	GenreTrance
	GenreClassical
	GenreInstrumental
	GenreAcid
	GenreHouse
	GenreGame
	GenreSoundClip
	GenreGospel
	GenreNoise
	GenreAlternativeRock
	GenreBass
	GenreSoul
	GenrePunk
	GenreSpace
	GenreMeditative
	GenreInstrumentalPop
	GenreInstrumentalRock
	GenreEthnic
	GenreGothic
	GenreDarkwave
	GenreTechnoindustrial
	GenreElectronic
	GenrePopFolk
	GenreEurodance
	GenreSouthernRock
	GenreComedy
	GenreCull
	GenreGangsta
	GenreTop40
	GenreChristianRap
	GenrePopSlashFunk
	GenreJungleMusic
	GenreNativeUS
	GenreCabaret
	GenreNewWave
	GenrePsychedelic
	GenreRave
	GenreShowtunes
	GenreTrailer
	GenreLofi
	GenreTribal
	GenreAcidPunk
	GenreAcidJazz
	GenrePolka
	GenreRetro
	GenreMusical
	GenreRockNRoll
	GenreHardRock
)

var resolveGenre = map[uint8]Genre{
	1: GenreBlues,
	2: GenreClassicRock,
	3: GenreCountry,
	4: GenreDance,
	5: GenreDisco,
	6: GenreFunk,
	7: GenreGrunge,
	8: GenreHipHop,
	9: GenreJazz,
	10: GenreMetal,
	11: GenreNewAge,
	12: GenreOldies,
	13: GenreOther,
	14: GenrePop,
	15: GenreRhythmAndBlues,
	16: GenreRap,
	17: GenreReggae,
	18: GenreRock,
	19: GenreTechno,
	20: GenreIndustrial,
	21: GenreAlternative,
	22: GenreSka,
	23: GenreDeathMetal,
	24: GenrePranks,
	25: GenreSoundtrack,
	26: GenreEurotechno,
	27: GenreAmbient,
	28: GenreTripHop,
	29: GenreVocal,
	30: GenreJassAndFunk,
	31: GenreFusion,
	32: GenreTrance,
	33: GenreClassical,
	34: GenreInstrumental,
	35: GenreAcid,
	36: GenreHouse,
	37: GenreGame,
	38: GenreSoundClip,
	39: GenreGospel,
	40: GenreNoise,
	41: GenreAlternativeRock,
	42: GenreBass,
	43: GenreSoul,
	44: GenrePunk,
	45: GenreSpace,
	46: GenreMeditative,
	47: GenreInstrumentalPop,
	48: GenreInstrumentalRock,
	49: GenreEthnic,
	50: GenreGothic,
	51: GenreDarkwave,
	52: GenreTechnoindustrial,
	53: GenreElectronic,
	54: GenrePopFolk,
	55: GenreEurodance,
	56: GenreSouthernRock,
	57: GenreComedy,
	58: GenreCull,
	59: GenreGangsta,
	60: GenreTop40,
	61: GenreChristianRap,
	62: GenrePopSlashFunk,
	63: GenreJungleMusic,
	64: GenreNativeUS,
	65: GenreCabaret,
	66: GenreNewWave,
	67: GenrePsychedelic,
	68: GenreRave,
	69: GenreShowtunes,
	70: GenreTrailer,
	71: GenreLofi,
	72: GenreTribal,
	73: GenreAcidPunk,
	74: GenreAcidJazz,
	75: GenrePolka,
	76: GenreRetro,
	77: GenreMusical,
	78: GenreRockNRoll,
	79: GenreHardRock,
}

var displayGenre = map[Genre]string{
	GenreBlues:            "Blues",
	GenreClassicRock:      "Classic Rock",
	GenreCountry:          "Country",
	GenreDance:            "Dance",
	GenreDisco:            "Disco",
	GenreFunk:             "Funk",
	GenreGrunge:           "Grunge",
	GenreHipHop:           "Hip Hop",
	GenreJazz:             "Jazz",
	GenreMetal:            "Metal",
	GenreNewAge:           "NewAge",
	GenreOldies:           "Oldies",
	GenreOther:            "Other",
	GenrePop:              "Pop",
	GenreRhythmAndBlues:   "Rhythm And Blues",
	GenreRap:              "Rap",
	GenreReggae:           "Reggae",
	GenreRock:             "Rock",
	GenreTechno:           "Techno",
	GenreIndustrial:       "Industrial",
	GenreAlternative:      "Alternative",
	GenreSka:              "Ska",
	GenreDeathMetal:       "Death Metal",
	GenrePranks:           "Pranks",
	GenreSoundtrack:       "Soundtrack",
	GenreEurotechno:       "Eurotechno",
	GenreAmbient:          "Ambient",
	GenreTripHop:          "TripHop",
	GenreVocal:            "Vocal",
	GenreJassAndFunk:      "Jass And Funk",
	GenreFusion:           "Fusion",
	GenreTrance:           "Trance",
	GenreClassical:        "Classical",
	GenreInstrumental:     "Instrumental",
	GenreAcid:             "Acid",
	GenreHouse:            "House",
	GenreGame:             "Game",
	GenreSoundClip:        "Sound Clip",
	GenreGospel:           "Gospel",
	GenreNoise:            "Noise",
	GenreAlternativeRock:  "Alternative Rock",
	GenreBass:             "Bass",
	GenreSoul:             "Soul",
	GenrePunk:             "Punk",
	GenreSpace:            "Space",
	GenreMeditative:       "Meditative",
	GenreInstrumentalPop:  "Instrumental Pop",
	GenreInstrumentalRock: "Instrumental Rock",
	GenreEthnic:           "Ethnic",
	GenreGothic:           "Gothic",
	GenreDarkwave:         "Darkwave",
	GenreTechnoindustrial: "Techno Industrial",
	GenreElectronic:       "Electronic",
	GenrePopFolk:          "Pop Folk",
	GenreEurodance:        "Eurodance",
	GenreSouthernRock:     "Southern Rock",
	GenreComedy:           "Comedy",
	GenreCull:             "Cull",
	GenreGangsta:          "Gangsta",
	GenreTop40:            "Top 40",
	GenreChristianRap:     "Christian Rap",
	GenrePopSlashFunk:     "Pop Slash Funk",
	GenreJungleMusic:      "Jungle Music",
	GenreNativeUS:         "Native US",
	GenreCabaret:          "Cabaret",
	GenreNewWave:          "NewWave",
	GenrePsychedelic:      "Psychedelic",
	GenreRave:             "Rave",
	GenreShowtunes:        "Showtunes",
	GenreTrailer:          "Trailer",
	GenreLofi:             "Lofi",
	GenreTribal:           "Tribal",
	GenreAcidPunk:         "Acid Punk",
	GenreAcidJazz:         "Acid Jazz",
	GenrePolka:            "Polka",
	GenreRetro:            "Retro",
	GenreMusical:          "Musical",
	GenreRockNRoll:        "RockNRoll",
	GenreHardRock:         "Hard Rock",
}

type MP4Picture struct {
	Format ImageType
	Data []byte
}

type MP4Tags struct {
	Album string // moov.udta.meta.ilst.(c)alb
	AlbumSort string
	AlbumArtist string // moov.udta.meta.ilst.aART
	AlbumArtistSort string
	Artist string // moov.udta.meta.ilst.(c)art
	ArtistSort string
	EncodingTool string // moov.udta.meta.ilst.(c)too
	BPM int16
	Comment string // moov.udta.meta.ilst.(c)cmt
	Composer string // moov.udta.meta.ilst.(c)wrt
	ComposerSort string
	Conductor string // moov.udta.meta.ilst.(c)con
	Copyright string // moov.udta.meta.ilst.cprt
	Custom map[string]string
	CustomGenre string // moov.udta.meta.ilst.(c)gen
	Date string // moov.udta.meta.ilst.(c)day
	Description string // moov.udta.meta.ilst.desc
	LongDescription string // moov.udta.meta.ilst.ldes
	Director string
	DiscNumber int16 // moov.udta.meta.ilst.disk
	DiscTotal int16 // moov.udta.meta.ilst.disk
	Genre Genre
	ItunesAdvisory ItunesAdvisory
	ItunesAlbumID int32
	ItunesArtistID int32
	ItunesStik ItunesStik // "moov.udta.meta.ilst.stik"
	Lyrics string // moov.udta.meta.ilst.(c)lyr
	Narrator string // moov.udta.meta.ilst.(c)nrt
	OtherCustom map[string][]string
	Pictures []*MP4Picture // "moov.udta.meta.ilst.covr"
	Publisher string // moov.udta.meta.ilst.(c)pub
	Title string // moov.udta.meta.ilst.(c)nam
	TitleSort string
	TrackNumber int16 // moov.udta.meta.ilst.trkn
	TrackTotal int16 // moov.udta.meta.ilst.trkn
	TVNetwork string // moov.udta.meta.ilst.tvnn
	TVShow string // moov.udta.meta.ilst.tvsh
	TVEpisode string // moov.udta.meta.ilst.tven
	TVEpisodeNum int16 // moov.udta.meta.ilst.tves
	TVSeason int16 // moov.udta.meta.ilst.tvsn
	Year int32
}