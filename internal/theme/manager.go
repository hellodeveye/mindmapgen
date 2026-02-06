package theme

import (
	"embed"
	"fmt"
	"path"
	"strings"
	"sync"

	"gopkg.in/yaml.v3"
)

//go:embed themes/*.yaml
var themesFS embed.FS

// Manager 主题管理器
type Manager struct {
	themes map[string]*ThemeConfig
	mu     sync.RWMutex
}

var (
	defaultManager *Manager
	once           sync.Once
)

// GetManager 获取全局主题管理器实例
func GetManager() *Manager {
	once.Do(func() {
		defaultManager = NewManager()
		if err := defaultManager.LoadEmbeddedThemes(); err != nil {
			// 如果加载失败，使用默认主题
			defaultManager.setDefaultTheme()
		}
	})
	return defaultManager
}

// NewManager 创建新的主题管理器
func NewManager() *Manager {
	return &Manager{
		themes: make(map[string]*ThemeConfig),
	}
}

// LoadEmbeddedThemes 加载嵌入的主题文件
func (m *Manager) LoadEmbeddedThemes() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	entries, err := themesFS.ReadDir("themes")
	if err != nil {
		return fmt.Errorf("failed to read themes directory: %w", err)
	}

	for _, entry := range entries {
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".yaml") {
			data, err := themesFS.ReadFile(path.Join("themes", entry.Name()))
			if err != nil {
				continue // 跳过无法读取的文件
			}

			var theme ThemeConfig
			if err := yaml.Unmarshal(data, &theme); err != nil {
				continue // 跳过无法解析的文件
			}

			// 使用文件名（不包含扩展名）作为主题ID
			themeID := entry.Name()[:len(entry.Name())-5] // 移除.yaml扩展名
			m.themes[themeID] = &theme
		}
	}

	// 如果没有加载到任何主题，设置默认主题
	if len(m.themes) == 0 {
		m.setDefaultTheme()
	}

	return nil
}

// GetTheme 获取指定主题
func (m *Manager) GetTheme(name string) (*ThemeConfig, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	theme, exists := m.themes[name]
	if !exists {
		// 如果主题不存在，返回默认主题
		if defaultTheme, hasDefault := m.themes["default"]; hasDefault {
			return defaultTheme, nil
		}
		return nil, fmt.Errorf("theme '%s' not found", name)
	}

	return theme, nil
}

// ListThemes 列出所有可用主题
func (m *Manager) ListThemes() []string {
	m.mu.RLock()
	defer m.mu.RUnlock()

	themes := make([]string, 0, len(m.themes))
	for name := range m.themes {
		themes = append(themes, name)
	}
	return themes
}

// setDefaultTheme 设置默认主题（硬编码）
func (m *Manager) setDefaultTheme() {
	defaultTheme := &ThemeConfig{
		Name:  "Default Theme",
		Style: "standard",
		Colors: ColorConfig{
			Background:     "#FFFFFF",
			ConnectionLine: "#0D0B22",
		},
		NodeStyles: NodeStylesConfig{
			Root: NodeStyleConfig{
				FillColor:   [3]float64{0.051, 0.043, 0.133},
				StrokeColor: [3]float64{0.051, 0.043, 0.133},
				TextColor:   [3]float64{1.0, 1.0, 1.0},
			},
			Level1: NodeStyleConfig{
				FillColor:   [3]float64{0.96, 0.97, 0.98},
				StrokeColor: [3]float64{0.96, 0.97, 0.98},
				TextColor:   [3]float64{0.0, 0.0, 0.0},
			},
			Level2: NodeStyleConfig{
				FillColor:   [3]float64{0.96, 0.97, 0.98},
				StrokeColor: [3]float64{0.96, 0.97, 0.98},
				TextColor:   [3]float64{0.0, 0.0, 0.0},
			},
			Leaf: NodeStyleConfig{
				FillColor:   [3]float64{1.0, 1.0, 1.0},
				StrokeColor: [3]float64{1.0, 1.0, 1.0},
				TextColor:   [3]float64{0.0, 0.0, 0.0},
			},
		},
		Layout: LayoutConfig{
			MinNodeWidth:  100.0,
			MaxNodeWidth:  240.0,
			MinNodeHeight: 36.0,
			LevelSpacing:  150.0,
			NodeSpacing:   30.0,
			CornerRadius:  8.0,
			FontSize:      15.0,
			Scale:         3.0,
			LineHeight:    20.0,
			TextPadding:   15.0,
		},
	}

	m.themes["default"] = defaultTheme
}
