package model

// ImageInfo 图集中单张图片信息
type ImageInfo struct {
	SubPageURL string // 子页面URL (如 pic2284_1.html)
	ThumbURL   string // 缩略图URL
	LargeURL   string // 大图URL (下载时填充)
	LocalPath  string // 本地保存路径 (下载时填充)
	ImageName  string // 文件名 (下载时填充)
	Index      int    // 在图集中的序号 (1-based)
}

// Album 图集数据模型
type Album struct {
	AlbumID   string
	Title     string
	DetailURL string
	CoverURL  string
	Tags      []string
	Images    []ImageInfo
}

func (a Album) ID() string {
	return a.AlbumID
}

func (a Album) IsPersistable() bool {
	return true
}
