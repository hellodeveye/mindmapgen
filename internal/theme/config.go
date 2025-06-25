package theme

import "github.com/hellodeveye/mindmapgen/pkg/types"

// ColorConfig 颜色配置
type ColorConfig struct {
	Background     string `yaml:"background"`
	ConnectionLine string `yaml:"connectionLine"`
}

// NodeStyleConfig 节点样式配置
type NodeStyleConfig struct {
	FillColor   [3]float64 `yaml:"fillColor"`
	StrokeColor [3]float64 `yaml:"strokeColor"`
	TextColor   [3]float64 `yaml:"textColor"`
}

// NodeStylesConfig 所有节点类型的样式配置
type NodeStylesConfig struct {
	Root   NodeStyleConfig `yaml:"root"`
	Level1 NodeStyleConfig `yaml:"level1"`
	Level2 NodeStyleConfig `yaml:"level2"`
	Leaf   NodeStyleConfig `yaml:"leaf"`
}

// SketchConfig 手绘风格配置
type SketchConfig struct {
	Roughness     float64 `yaml:"roughness"`     // 抖动强度 (0-10)
	Iterations    int     `yaml:"iterations"`    // 描边次数 (1-5)
	LineVariation float64 `yaml:"lineVariation"` // 线条变化度 (0-5)
	FillPattern   string  `yaml:"fillPattern"`   // 填充图案: none, dots, crosshatch
	Seed          int64   `yaml:"seed"`          // 随机种子，确保一致性
}

// LayoutConfig 布局配置
type LayoutConfig struct {
	MinNodeWidth  float64 `yaml:"minNodeWidth"`
	MaxNodeWidth  float64 `yaml:"maxNodeWidth"`
	MinNodeHeight float64 `yaml:"minNodeHeight"`
	LevelSpacing  float64 `yaml:"levelSpacing"`
	NodeSpacing   float64 `yaml:"nodeSpacing"`
	CornerRadius  float64 `yaml:"cornerRadius"`
	FontSize      float64 `yaml:"fontSize"`
	Scale         float64 `yaml:"scale"`
	LineHeight    float64 `yaml:"lineHeight"`
	TextPadding   float64 `yaml:"textPadding"`
}

// ThemeConfig 主题配置
type ThemeConfig struct {
	Name         string           `yaml:"name"`
	Style        string           `yaml:"style"` // "standard" 或 "sketch"
	Colors       ColorConfig      `yaml:"colors"`
	NodeStyles   NodeStylesConfig `yaml:"nodeStyles"`
	Layout       LayoutConfig     `yaml:"layout"`
	SketchConfig *SketchConfig    `yaml:"sketchConfig,omitempty"` // 仅手绘风格需要
}

// ToNodeStyle 将配置转换为NodeStyle结构
func (nsc NodeStyleConfig) ToNodeStyle() *types.NodeStyle {
	return &types.NodeStyle{
		FillColor:   nsc.FillColor,
		StrokeColor: nsc.StrokeColor,
		TextColor:   nsc.TextColor,
	}
}

// GetNodeStyles 获取所有节点样式
func (tc *ThemeConfig) GetNodeStyles() map[string]*types.NodeStyle {
	return map[string]*types.NodeStyle{
		"root":   tc.NodeStyles.Root.ToNodeStyle(),
		"level1": tc.NodeStyles.Level1.ToNodeStyle(),
		"level2": tc.NodeStyles.Level2.ToNodeStyle(),
		"leaf":   tc.NodeStyles.Leaf.ToNodeStyle(),
	}
}

// IsSketchStyle 判断是否为手绘风格
func (tc *ThemeConfig) IsSketchStyle() bool {
	return tc.Style == "sketch" && tc.SketchConfig != nil
}
