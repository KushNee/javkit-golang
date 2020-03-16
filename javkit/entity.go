package javkit

// Config 用于对相关的 ini 配置文件反序列化
type Config struct {
	IfNfo                 string
	IfExnfo               string
	IfReview              string
	CustomTitle           string
	CustomSubtitle        string
	IfMP4                 string
	RenameMP4             string
	IfFolder              string
	RenameFolder          string
	IfClassify            string
	IgnoreParentErr       string
	ClassifyRoot          string
	ClassifyBasis         string
	CreateSymboliclink    string
	SymboliclinkDirectory string
	IfJpg                 string
	CustomFanart          string
	CustomPoster          string
	IfSculpture           string
	IfProxy               string
	Proxy                 string
	IfPlot                string
	IfTran                string
	TransId               string
	TransSk               string
	SimpTrad              string
	LibraryUrl            string
	BusUrl                string
	SurenPref             string
	FileType              string
	TitleLen              int
	Script                string
}

// JavFile	每部 jav 的结构体
type JavFile struct {
	Path     string // 视频文件名称
	License  string // 车牌-番号
	Episodes int    // 集数

}

type JavInfo struct {
	License       string
	LicensePrefix string
	Title         string
	FullTitle     string
	Director      string
	Release       JavReleaseDate
	Studio        string
	Score         string
	FirstActress  string
	AllActress    []string
	Length        int
	ChineseSub    bool
	VideoName     string
	CoverUrl      string
	Review        string // 精彩评论
	Introduction  string // 作品介绍
	Genres        []string
}

type JavReleaseDate struct {
	Year     string
	Month    string
	Day      string
	FullDate string
}
