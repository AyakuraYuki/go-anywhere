package handler

import (
	"fmt"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"golang.org/x/text/collate"
	"golang.org/x/text/language"
)

const (
	Byte = 1
	KB   = Byte * 1024
	MB   = KB * 1024
	GB   = MB * 1024
	TB   = GB * 1024
)

type FileInfo struct {
	Name    string
	URL     string
	Size    string
	ModTime string
	IsDir   bool
	Icon    string
}

type sortableFile struct {
	FileInfo

	basename string
	ext      string

	basenameKey []byte
	extKey      []byte
}

type DirListData struct {
	Path      string
	Parent    string
	HasParent bool
	Files     []FileInfo
}

func formatSize(size int64) string {
	switch {
	case size >= TB:
		return fmt.Sprintf("%.2f TB", float64(size)/float64(TB))
	case size >= GB:
		return fmt.Sprintf("%.2f GB", float64(size)/float64(GB))
	case size >= MB:
		return fmt.Sprintf("%.2f MB", float64(size)/float64(MB))
	case size >= KB:
		return fmt.Sprintf("%.2f KB", float64(size)/float64(KB))
	case size > Byte:
		return fmt.Sprintf("%d Bytes", size)
	default:
		return fmt.Sprintf("%d Byte", size)
	}
}

func getIcon(name string, isDir bool) string {
	if isDir {
		return "📁"
	}

	ext := strings.ToLower(filepath.Ext(name))
	switch ext {
	case ".html", ".htm":
		return "🌐"
	case ".css":
		return "🎨"
	case ".js", ".ts", ".jsx", ".tsx":
		return "📜"
	case ".go", ".py", ".java", ".c", ".cpp", ".rs", ".rb":
		return "💻"
	case ".json", ".yaml", ".yml", ".toml", ".xml":
		return "📋"
	case ".md", ".txt", ".doc", ".docx", ".pdf":
		return "📄"
	case ".jpg", ".jpeg", ".png", ".gif", ".svg", ".webp", ".ico", ".bmp":
		return "🖼️"
	case ".mp4", ".avi", ".mov", ".mkv", ".webm":
		return "🎬"
	case ".mp3", ".wav", ".flac", ".ogg", ".aac":
		return "🎵"
	case ".zip", ".tar", ".gz", ".rar", ".7z", ".bz2":
		return "📦"
	case ".exe", ".bin", ".sh", ".bat":
		return "⚙️"
	default:
		return "📄"
	}
}

func sortFiles(files []FileInfo, extEmptyLast bool) {
	collator := collate.New(language.Und, collate.IgnoreCase)
	buf := collate.Buffer{}

	sortableFiles := make([]sortableFile, len(files))
	for i, file := range files {
		basename := filepath.Base(file.Name)
		ext := strings.ToLower(filepath.Ext(file.Name))

		sortableFiles[i] = sortableFile{
			FileInfo:    file,
			basename:    basename,
			ext:         ext,
			basenameKey: collator.KeyFromString(&buf, basename),
			extKey:      collator.KeyFromString(&buf, ext),
		}
	}

	sort.SliceStable(sortableFiles, func(i, j int) bool {
		a := sortableFiles[i]
		b := sortableFiles[j]

		// directory first
		if a.IsDir != b.IsDir {
			return a.IsDir
		}

		// directories in alphabetical
		if a.IsDir {
			return collator.Compare(a.basenameKey, b.basenameKey) < 0
		}

		// empty extension to last
		if extEmptyLast {
			if a.ext == "" && b.ext != "" {
				return false
			}
			if a.ext != "" && b.ext == "" {
				return true
			}
		}

		// extensions in alphabetical
		if c := collator.Compare(a.extKey, b.extKey); c != 0 {
			return c < 0
		}

		// filename in alphabetical
		return collator.Compare(a.basenameKey, b.basenameKey) < 0
	})

	for i := range files {
		files[i] = sortableFiles[i].FileInfo
	}
}

func BuildDirListData(dirPath, urlPath string) (*DirListData, error) {
	entries, err := os.ReadDir(dirPath)
	if err != nil {
		return nil, err
	}

	var (
		files      []FileInfo
		parentPath string
		hasParent  = urlPath != "/" && urlPath != ""
	)

	for _, entry := range entries {
		// skip hidden entries
		if strings.HasPrefix(entry.Name(), ".") {
			continue
		}

		info, err := entry.Info()
		if err != nil {
			continue
		}

		var (
			name        = entry.Name()
			displayName = entry.Name()
			fileURL     = path.Join(urlPath, url.PathEscape(name))
			sizeStr     = ""
		)

		if entry.IsDir() {
			displayName = displayName + "/"
			fileURL = fileURL + "/"
			sizeStr = "-"
		} else {
			sizeStr = formatSize(info.Size())
		}

		files = append(files, FileInfo{
			Name:    displayName,
			URL:     fileURL,
			Size:    sizeStr,
			ModTime: info.ModTime().Format(time.DateTime),
			IsDir:   entry.IsDir(),
			Icon:    getIcon(name, entry.IsDir()),
		})
	}

	// sort files
	sortFiles(files, true)

	if hasParent {
		parentPath = path.Dir(strings.TrimSuffix(urlPath, "/"))
		if parentPath != "/" {
			parentPath += "/"
		}
	}

	return &DirListData{
		Path:      urlPath,
		Parent:    parentPath,
		HasParent: hasParent,
		Files:     files,
	}, nil
}
