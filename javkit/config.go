package javkit

// TomlConfig 用于对相关的 toml 配置文件反序列化
type TomlConfig struct {
	ColletNfo struct {
		SkipFolderWithNfo bool   `toml:"skipFolderWithNfo"`
		CollectNfo        bool   `toml:"collectNfo"`
		CollectReview     bool   `toml:"collectReview"`
		TitleStyle        string `toml:"titleStyle"`
		ChineseSubStyle   string `toml:"chineseSubStyle"`
	} `toml:"colletNfo"`
	MoveVideo struct {
		RenameVideo   bool   `toml:"renameVideo"`
		RenamePattern string `toml:"renamePattern"`
	} `toml:"moveVideo"`
	ModifyFolder struct {
		Mkdir         bool   `toml:"mkdir"`
		FolderPattern string `toml:"folderPattern"`
	} `toml:"modifyFolder"`
	Archive struct {
		Archive              bool   `toml:"archive"`
		IgnoreUpperPathError bool   `toml:"ignoreUpperPathError"`
		RootPath             string `toml:"rootPath"`
		Symlink              bool   `toml:"symlink"`
		ArchivePattern       string `toml:"archivePattern"`
		SymlinkPath          string `toml:"symlinkPath"`
	} `toml:"archive"`
	Introduction struct {
		CollectIntroduction bool `toml:"collectIntroduction"`
	} `toml:introduction`
	Cover struct {
		DownloadCover bool   `toml:"downloadCover"`
		FanartPattern string `toml:"fanartPattern"`
		PosterPattern string `toml:"posterPattern"`
	} `toml:"cover"`
	Proxy struct {
		UseProxy bool   `toml:"useProxy"`
		ProxyURL string `toml:"proxyUrl"`
	} `toml:"proxy"`
	Other struct {
		SimpleTradition   string `toml:"simpleTradition"`
		Javlibrary        string `toml:"javlibrary"`
		Javbus            string `toml:"javbus"`
		Suren             string `toml:"suren"`
		RenameTitleLength int    `toml:"renameTitleLength"`
		PythonScript      string `toml:"pythonScript"`
	} `toml:"other"`
}

// IniConfig 用于对相关的 ini 配置文件反序列化
type IniConfig struct {
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
	Interpreter           string
}
