package javkit

import "encoding/xml"

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

type MovieInfo struct {
	XMLName      xml.Name `xml:"movie"`
	Text         string   `xml:",chardata"`
	Plot         string   `xml:"plot"`
	Outline      string   `xml:"outline"`
	Customrating string   `xml:"customrating"`
	Lockdata     string   `xml:"lockdata"`
	Dateadded    string   `xml:"dateadded"`
	Title        string   `xml:"title"`
	Director     string   `xml:"director"`
	Rating       string   `xml:"rating"`
	Year         string   `xml:"year"`
	Mpaa         string   `xml:"mpaa"`
	Imdbid       string   `xml:"imdbid"`
	Countrycode  string   `xml:"countrycode"`
	Premiered    string   `xml:"premiered"`
	Releasedate  string   `xml:"releasedate"`
	Criticrating string   `xml:"criticrating"`
	Runtime      string   `xml:"runtime"`
	Country      string   `xml:"country"`
	GenreList    []string `xml:"genre"`
	Studio       string   `xml:"studio"`
	TagList      []string `xml:"tag"`
	Art          struct {
		Text   string   `xml:",chardata"`
		Poster string   `xml:"poster"`
		Fanart []string `xml:"fanart"`
	} `xml:"art"`
	Isuserfavorite string `xml:"isuserfavorite"`
	Playcount      string `xml:"playcount"`
	Watched        string `xml:"watched"`
	Resume         struct {
		Text     string `xml:",chardata"`
		Position string `xml:"position"`
		Total    string `xml:"total"`
	} `xml:"resume"`
	ActorList []actor  `xml:"actor"`
	ID        string   `xml:"id"`
	Fileinfo  fileInfo `xml:"fileinfo"`
	Release   string   `xml:"release"`
	Num       string   `xml:"num"`
}

type actor struct {
	XMLName xml.Name `xml:"actor"`
	Text    string   `xml:",chardata"`
	Name    string   `xml:"name"`
	Type    string   `xml:"type"`
	Thumb   string   `xml:"thumb"`
}

type fileInfo struct {
	XMLName       xml.Name `xml:"fileinfo"`
	Text          string   `xml:",chardata"`
	Streamdetails struct {
		Text  string `xml:",chardata"`
		Video struct {
			Text              string `xml:",chardata"`
			Codec             string `xml:"codec"`
			Micodec           string `xml:"micodec"`
			Bitrate           string `xml:"bitrate"`
			Width             string `xml:"width"`
			Height            string `xml:"height"`
			Aspect            string `xml:"aspect"`
			Aspectratio       string `xml:"aspectratio"`
			Framerate         string `xml:"framerate"`
			Language          string `xml:"language"`
			Scantype          string `xml:"scantype"`
			Default           string `xml:"default"`
			Forced            string `xml:"forced"`
			Duration          string `xml:"duration"`
			Durationinseconds string `xml:"durationinseconds"`
		} `xml:"video"`
		Audio struct {
			Text         string `xml:",chardata"`
			Codec        string `xml:"codec"`
			Micodec      string `xml:"micodec"`
			Bitrate      string `xml:"bitrate"`
			Language     string `xml:"language"`
			Scantype     string `xml:"scantype"`
			Channels     string `xml:"channels"`
			Samplingrate string `xml:"samplingrate"`
			Default      string `xml:"default"`
			Forced       string `xml:"forced"`
		} `xml:"audio"`
	} `xml:"streamdetails"`
}
