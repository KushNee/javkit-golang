package javkit

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
