package drawer

import (
	_ "embed" // Ensure embed is imported for //go:embed
	"fmt"
	"io"
	"math"
	"os"
	"path/filepath"
	"strconv"
	"unicode"

	"github.com/fogleman/gg"
	"github.com/hellodeveye/mindmapgen/internal/theme"
	"github.com/hellodeveye/mindmapgen/pkg/types"
)

//go:embed fonts/simhei.ttf
var simhei []byte

type embeddedFont struct {
	Name string
	Data []byte
}

var embeddedFonts = []embeddedFont{
	{"simhei.ttf", simhei},
}

// 默认常量 - 现在从主题配置中获取
const (
	DefaultMinNodeWidth  = 100.0
	DefaultMaxNodeWidth  = 240.0
	DefaultMinNodeHeight = 36.0
	DefaultLevelSpacing  = 150.0
	DefaultNodeSpacing   = 30.0
	DefaultCornerRadius  = 8.0
	DefaultFontSize      = 15.0
	DefaultScale         = 3.0
	DefaultLineHeight    = 20.0
	DefaultTextPadding   = 15.0
)

type Bounds struct {
	MinX, MinY, MaxX, MaxY float64
}

// 存储节点尺寸的结构
type NodeSize struct {
	Width           float64
	Height          float64
	Lines           []string // 存储换行后的文本
	ActualTextWidth float64
}

// DrawConfig 绘制配置
type DrawConfig struct {
	Theme               *theme.ThemeConfig
	MinNodeWidth        float64
	MaxNodeWidth        float64
	MinNodeHeight       float64
	LevelSpacing        float64
	NodeSpacing         float64
	CornerRadius        float64
	FontSize            float64
	Scale               float64
	LineHeight          float64
	TextPadding         float64
	BackgroundColor     [3]float64
	ConnectionLineColor [3]float64
}

// NewDrawConfig 根据主题创建绘制配置
func NewDrawConfig(themeName string) (*DrawConfig, error) {
	manager := theme.GetManager()
	themeConfig, err := manager.GetTheme(themeName)
	if err != nil {
		return nil, err
	}

	// 解析背景颜色
	bgColor := parseHexColor(themeConfig.Colors.Background, [3]float64{1.0, 1.0, 1.0})
	lineColor := parseHexColor(themeConfig.Colors.ConnectionLine, [3]float64{0.051, 0.043, 0.133})

	return &DrawConfig{
		Theme:               themeConfig,
		MinNodeWidth:        themeConfig.Layout.MinNodeWidth,
		MaxNodeWidth:        themeConfig.Layout.MaxNodeWidth,
		MinNodeHeight:       themeConfig.Layout.MinNodeHeight,
		LevelSpacing:        themeConfig.Layout.LevelSpacing,
		NodeSpacing:         themeConfig.Layout.NodeSpacing,
		CornerRadius:        themeConfig.Layout.CornerRadius,
		FontSize:            themeConfig.Layout.FontSize,
		Scale:               themeConfig.Layout.Scale,
		LineHeight:          themeConfig.Layout.LineHeight,
		TextPadding:         themeConfig.Layout.TextPadding,
		BackgroundColor:     bgColor,
		ConnectionLineColor: lineColor,
	}, nil
}

// parseHexColor 解析十六进制颜色为RGB数组
func parseHexColor(hex string, defaultColor [3]float64) [3]float64 {
	if hex == "" || hex[0] != '#' || len(hex) != 7 {
		return defaultColor
	}

	r, err1 := strconv.ParseInt(hex[1:3], 16, 0)
	g, err2 := strconv.ParseInt(hex[3:5], 16, 0)
	b, err3 := strconv.ParseInt(hex[5:7], 16, 0)

	if err1 != nil || err2 != nil || err3 != nil {
		return defaultColor
	}

	return [3]float64{float64(r) / 255.0, float64(g) / 255.0, float64(b) / 255.0}
}

func loadFont(dc *gg.Context, size float64) error {
	fontLoaded := false

	for _, font := range embeddedFonts {
		fontBytes := font.Data
		if len(fontBytes) == 0 {
			continue
		}

		suffix := filepath.Ext(font.Name)
		if suffix == "" {
			suffix = ".font"
		}
		tmpfile, err := os.CreateTemp("", fmt.Sprintf("font*%s", suffix))
		if err != nil {
			fmt.Printf("Warning: failed to create temporary font file for %s: %v\n", font.Name, err)
			continue
		}
		tmpFileName := tmpfile.Name()
		defer os.Remove(tmpFileName)

		if _, err := tmpfile.Write(fontBytes); err != nil {
			fmt.Printf("Warning: failed to write to temporary font file %s: %v\n", tmpFileName, err)
			tmpfile.Close()
			continue
		}

		if err := tmpfile.Close(); err != nil {
			fmt.Printf("Warning: failed to close temporary font file %s: %v\n", tmpFileName, err)
			continue
		}

		if err := dc.LoadFontFace(tmpFileName, size); err == nil {
			fontLoaded = true
			break
		} else {
			fmt.Printf("Warning: failed to load font from temp file %s: %v\n", tmpFileName, err)
		}
	}

	if !fontLoaded {
		dc.LoadFontFace("", size)
		return fmt.Errorf("failed to load preferred fonts from embed, using default font")
	}

	return nil
}

// 保存对根节点的引用，用于识别根节点
var root *types.Node

// Draw 使用默认主题绘制思维导图
func Draw(rootNode *types.Node, w io.Writer) error {
	return DrawWithTheme(rootNode, w, "default")
}

// DrawWithTheme 使用指定主题绘制思维导图
func DrawWithTheme(rootNode *types.Node, w io.Writer, themeName string) error {
	config, err := NewDrawConfig(themeName)
	if err != nil {
		// 如果主题加载失败，使用默认配置
		config = &DrawConfig{
			MinNodeWidth:        DefaultMinNodeWidth,
			MaxNodeWidth:        DefaultMaxNodeWidth,
			MinNodeHeight:       DefaultMinNodeHeight,
			LevelSpacing:        DefaultLevelSpacing,
			NodeSpacing:         DefaultNodeSpacing,
			CornerRadius:        DefaultCornerRadius,
			FontSize:            DefaultFontSize,
			Scale:               DefaultScale,
			LineHeight:          DefaultLineHeight,
			TextPadding:         DefaultTextPadding,
			BackgroundColor:     [3]float64{1.0, 1.0, 1.0},
			ConnectionLineColor: [3]float64{0.051, 0.043, 0.133},
		}
	}

	// 创建临时上下文用于文本测量
	tempDC := gg.NewContext(1, 1)
	if err := loadFont(tempDC, config.FontSize); err != nil {
		fmt.Printf("Warning: %v\n", err)
	}

	// 计算节点尺寸
	nodeSizes := make(map[*types.Node]*NodeSize)
	calculateNodeSizes(tempDC, rootNode, nodeSizes, config)

	// 获取树的深度和每层节点数
	maxDepth := 0
	levelCounts := make(map[int]int)
	calculateTreeMetrics(rootNode, 0, &maxDepth, levelCounts)

	// 保存根节点引用
	root = rootNode

	// 计算水平思维导图布局
	subtreeHeights := make(map[*types.Node]float64)
	calculateSubtreeHeights(rootNode, nodeSizes, subtreeHeights, config)
	horizontalMindmapLayout(rootNode, 0, 0, nodeSizes, subtreeHeights, config)

	// 计算边界
	bounds := &Bounds{
		MinX: math.MaxFloat64,
		MinY: math.MaxFloat64,
		MaxX: -math.MaxFloat64,
		MaxY: -math.MaxFloat64,
	}
	calculateBoundsWithSizes(rootNode, nodeSizes, bounds)

	// 扩展边界，确保有足够的边距
	extraMargin := 50.0
	bounds.MinX -= extraMargin
	bounds.MinY -= extraMargin
	bounds.MaxX += extraMargin
	bounds.MaxY += extraMargin

	// 计算画布尺寸
	contentWidth := bounds.MaxX - bounds.MinX
	contentHeight := bounds.MaxY - bounds.MinY
	canvasWidth := contentWidth
	canvasHeight := contentHeight

	// 创建最终上下文
	dc := gg.NewContext(int(canvasWidth*config.Scale), int(canvasHeight*config.Scale))
	dc.SetLineWidth(1.0 * config.Scale)
	dc.SetLineJoin(gg.LineJoinRound)
	dc.SetLineCap(gg.LineCapButt)

	if err := loadFont(dc, config.FontSize*config.Scale); err != nil {
		fmt.Printf("Warning: %v\n", err)
	}

	// 设置背景
	dc.SetRGB(config.BackgroundColor[0], config.BackgroundColor[1], config.BackgroundColor[2])
	dc.Clear()

	// 应用变换
	dc.Translate(-bounds.MinX*config.Scale, -bounds.MinY*config.Scale)

	// 先绘制所有连接线
	drawConnectionsHorizontal(dc, rootNode, nodeSizes, config)

	// 然后绘制所有节点
	drawAllNodes(dc, rootNode, nodeSizes, config)

	return dc.EncodePNG(w)
}

// 计算每个节点及其子树所需的总垂直高度
func calculateSubtreeHeights(node *types.Node, nodeSizes map[*types.Node]*NodeSize, subtreeHeights map[*types.Node]float64, config *DrawConfig) {
	if node == nil {
		return
	}

	nodeSize := nodeSizes[node]
	if nodeSize == nil {
		return
	}

	if len(node.Children) == 0 {
		subtreeHeights[node] = nodeSize.Height
		return
	}

	totalChildrenHeight := 0.0
	for _, child := range node.Children {
		calculateSubtreeHeights(child, nodeSizes, subtreeHeights, config)
		totalChildrenHeight += subtreeHeights[child]
	}

	// 加上节点间的垂直间距
	totalChildrenHeight += config.NodeSpacing * float64(len(node.Children)-1)

	// 子树高度是自身高度和子节点总高度中的较大值
	subtreeHeights[node] = math.Max(nodeSize.Height, totalChildrenHeight)
}

// 水平思维导图布局算法
func horizontalMindmapLayout(node *types.Node, x, y float64, nodeSizes map[*types.Node]*NodeSize, subtreeHeights map[*types.Node]float64, config *DrawConfig) {
	if node == nil {
		return
	}

	nodeSize := nodeSizes[node]
	if nodeSize == nil {
		return
	}

	// 设置当前节点位置 (中心点)
	node.X = x + nodeSize.Width/2
	node.Y = y

	// 如果没有子节点，结束递归
	if len(node.Children) == 0 {
		return
	}

	// 计算子节点起始垂直位置
	childrenTotalHeight := 0.0
	for _, child := range node.Children {
		childrenTotalHeight += subtreeHeights[child]
	}
	childrenTotalHeight += config.NodeSpacing * float64(len(node.Children)-1)

	currentY := y - childrenTotalHeight/2

	// 子节点的水平位置
	childX := x + nodeSize.Width + config.LevelSpacing

	// 递归放置子节点
	for _, child := range node.Children {
		childSubtreeHeight := subtreeHeights[child]
		// 将子节点垂直居中在其子树所占空间内
		childY := currentY + childSubtreeHeight/2

		horizontalMindmapLayout(child, childX, childY, nodeSizes, subtreeHeights, config)

		// 更新下一个子节点的起始Y坐标
		currentY += childSubtreeHeight + config.NodeSpacing
	}
}

// 绘制水平布局的连接线
func drawConnectionsHorizontal(dc *gg.Context, node *types.Node, nodeSizes map[*types.Node]*NodeSize, config *DrawConfig) {
	if node == nil || len(node.Children) == 0 {
		return
	}

	parentSize := nodeSizes[node]
	if parentSize == nil {
		return
	}

	// 连接起点（父节点右侧中心）
	startX := (node.X + parentSize.Width/2) * config.Scale
	startY := node.Y * config.Scale

	for _, child := range node.Children {
		childSize := nodeSizes[child]
		if childSize == nil {
			continue
		}

		// 连接终点（子节点左侧中心）
		endX := (child.X - childSize.Width/2) * config.Scale
		endY := child.Y * config.Scale

		if len(child.Children) == 0 { // 是叶子节点
			// 对于叶子节点，连接线应在文本开始前停止
			// 文本在 child.X 处水平居中
			textLeftEdgeX := child.X - childSize.ActualTextWidth/2
			textGap := 5.0 // 线条与文本的间隙
			endX = (textLeftEdgeX - textGap) * config.Scale
		}

		// 设置连接线样式
		dc.SetRGB(config.ConnectionLineColor[0], config.ConnectionLineColor[1], config.ConnectionLineColor[2])
		dc.SetLineWidth(1.0 * config.Scale)

		// 绘制平滑的S形连接线 (Bézier curve)
		dc.MoveTo(startX, startY)
		controlX1 := startX + (endX-startX)/2
		controlY1 := startY
		controlX2 := startX + (endX-startX)/2
		controlY2 := endY
		dc.CubicTo(controlX1, controlY1, controlX2, controlY2, endX, endY)
		dc.Stroke()

		// 递归绘制子节点的连接线
		drawConnectionsHorizontal(dc, child, nodeSizes, config)
	}
}

// 绘制单个节点
func drawSingleNode(dc *gg.Context, node *types.Node, isRoot bool, nodeSizes map[*types.Node]*NodeSize, scale float64, config *DrawConfig) {
	if node == nil {
		return
	}

	style := getNodeStyle(node, isRoot, config)
	nodeSize := nodeSizes[node]

	if nodeSize == nil {
		return
	}

	// 计算节点位置
	x := (node.X - nodeSize.Width/2) * scale
	y := (node.Y - nodeSize.Height/2) * scale
	w := nodeSize.Width * scale
	h := nodeSize.Height * scale
	r := config.CornerRadius * scale

	// 绘制节点背景
	dc.SetRGB(style.FillColor[0], style.FillColor[1], style.FillColor[2])
	drawRoundedRect(dc, x, y, w, h, r)
	dc.Fill()

	// 绘制节点边框
	dc.SetRGB(style.StrokeColor[0], style.StrokeColor[1], style.StrokeColor[2])
	dc.SetLineWidth(0.8 * scale)
	drawRoundedRect(dc, x, y, w, h, r)
	dc.Stroke()

	// 绘制文本
	dc.SetRGB(style.TextColor[0], style.TextColor[1], style.TextColor[2])
	scaledLineHeight := config.LineHeight * scale
	startY := (node.Y * scale) - (float64(len(nodeSize.Lines))*scaledLineHeight)/2 + scaledLineHeight/2

	for i, line := range nodeSize.Lines {
		y := startY + float64(i)*scaledLineHeight
		dc.DrawStringAnchored(line, node.X*scale, y, 0.5, 0.5)
	}
}

func calculateNodeSizes(dc *gg.Context, node *types.Node, nodeSizes map[*types.Node]*NodeSize, config *DrawConfig) {
	if node == nil {
		return
	}

	// 计算当前节点的尺寸，其宽度仅由其自身文本决定
	size := calculateTextWrapping(dc, node.Text, config)
	nodeSizes[node] = size

	// 递归为所有子节点计算尺寸
	for _, child := range node.Children {
		calculateNodeSizes(dc, child, nodeSizes, config)
	}
}

// 修改计算文本换行和节点尺寸的函数，提高效率和美观度
func calculateTextWrapping(dc *gg.Context, text string, config *DrawConfig) *NodeSize {
	words := splitIntoWords(text)
	if len(words) == 0 {
		return &NodeSize{Width: config.MinNodeWidth, Height: config.MinNodeHeight, ActualTextWidth: 0}
	}

	// 计算单行文本宽度
	textWidth := 0.0
	for _, word := range words {
		w, _ := dc.MeasureString(word)
		textWidth += w
	}
	spaceW, _ := dc.MeasureString(" ")
	textWidth += float64(len(words)-1) * spaceW

	// 添加文本内边距
	nodeWidth := textWidth + 2*config.TextPadding

	// 确保节点宽度在限制范围内
	if nodeWidth < config.MinNodeWidth {
		nodeWidth = config.MinNodeWidth
	} else if nodeWidth > config.MaxNodeWidth {
		nodeWidth = config.MaxNodeWidth
	}

	// 使用简化的换行策略
	availableWidth := nodeWidth - 2*config.TextPadding
	lines := breakTextIntoLines(dc, words, availableWidth)

	// 检查是否存在非常长的行，如果有，对这些行再次进行拆分
	var finalLines []string
	maxLineChars := 20 // 中文字符的最大行字符数

	for _, line := range lines {
		// 计算中文字符的数量
		chineseCount := 0
		for _, r := range line {
			if unicode.Is(unicode.Han, r) {
				chineseCount++
			}
		}

		// 如果一行中的中文字符数量过多，尝试在中文字符之间强制换行
		if chineseCount > maxLineChars {
			// 将长行分成更短的段落
			var parts []string
			var currentPart string
			count := 0

			for _, r := range line {
				currentPart += string(r)
				if unicode.Is(unicode.Han, r) {
					count++
					// 每10个中文字符左右进行换行
					if count >= 10 {
						parts = append(parts, currentPart)
						currentPart = ""
						count = 0
					}
				}
			}

			if currentPart != "" {
				parts = append(parts, currentPart)
			}

			finalLines = append(finalLines, parts...)
		} else {
			finalLines = append(finalLines, line)
		}
	}

	var maxLineWidth float64
	for _, line := range finalLines {
		w, _ := dc.MeasureString(line)
		if w > maxLineWidth {
			maxLineWidth = w
		}
	}

	// 计算节点高度
	nodeHeight := float64(len(finalLines))*config.LineHeight + 2*config.TextPadding
	if nodeHeight < config.MinNodeHeight {
		nodeHeight = config.MinNodeHeight
	}

	return &NodeSize{
		Width:           nodeWidth,
		Height:          nodeHeight,
		Lines:           finalLines,
		ActualTextWidth: maxLineWidth,
	}
}

// 新增一个辅助函数用于文本换行
func breakTextIntoLines(dc *gg.Context, words []string, availableWidth float64) []string {
	var lines []string
	currentLine := ""
	currentWidth := 0.0

	for i, word := range words {
		wordWidth, _ := dc.MeasureString(word)
		spaceWidth := 0.0
		if i > 0 && currentLine != "" {
			spaceWidth, _ = dc.MeasureString(" ")
		}

		// 检查是否需要换行
		if currentWidth+wordWidth+spaceWidth <= availableWidth {
			if currentLine != "" {
				currentLine += " "
				currentWidth += spaceWidth
			}
			currentLine += word
			currentWidth += wordWidth
		} else {
			if currentLine != "" {
				lines = append(lines, currentLine)
			}
			currentLine = word
			currentWidth = wordWidth
		}
	}

	if currentLine != "" {
		lines = append(lines, currentLine)
	}

	return lines
}

// 将文本分割成词（考虑中英文混合的情况） - 优化中文处理
func splitIntoWords(text string) []string {
	var words []string
	var currentWord []rune
	var inChineseSequence bool // 跟踪是否在连续的中文字符序列中

	for _, r := range text {
		isHan := unicode.Is(unicode.Han, r)
		isSpace := unicode.IsSpace(r)

		if isSpace {
			// 遇到空格，结束当前单词（无论是中文还是英文）
			if len(currentWord) > 0 {
				words = append(words, string(currentWord))
				currentWord = nil
			}
			inChineseSequence = false
		} else if isHan {
			// 如果是从非中文切换到中文
			if !inChineseSequence && len(currentWord) > 0 {
				words = append(words, string(currentWord))
				currentWord = nil
			}
			// 添加当前中文字符到序列
			currentWord = append(currentWord, r)
			inChineseSequence = true
		} else {
			// 如果是从中文切换到非中文
			if inChineseSequence && len(currentWord) > 0 {
				words = append(words, string(currentWord))
				currentWord = nil
			}
			// 添加当前非中文字符
			currentWord = append(currentWord, r)
			inChineseSequence = false
		}
	}

	// 保存最后累积的单词
	if len(currentWord) > 0 {
		words = append(words, string(currentWord))
	}

	return words
}

func calculateTreeMetrics(node *types.Node, level int, maxDepth *int, levelCounts map[int]int) {
	if node == nil {
		return
	}

	// 更新最大深度
	if level > *maxDepth {
		*maxDepth = level
	}

	// 更新当前层的节点数
	levelCounts[level]++

	// 递归处理子节点
	for _, child := range node.Children {
		calculateTreeMetrics(child, level+1, maxDepth, levelCounts)
	}
}

// 绘制所有节点（与连接线分离，确保节点绘制在连接线上方）
func drawAllNodes(dc *gg.Context, node *types.Node, nodeSizes map[*types.Node]*NodeSize, config *DrawConfig) {
	if node == nil {
		return
	}

	// 绘制当前节点
	drawSingleNode(dc, node, node == root, nodeSizes, config.Scale, config)

	// 递归处理所有子节点
	for _, child := range node.Children {
		drawAllNodes(dc, child, nodeSizes, config)
	}
}

func calculateBoundsWithSizes(node *types.Node, nodeSizes map[*types.Node]*NodeSize, bounds *Bounds) {
	if node == nil {
		return
	}

	size := nodeSizes[node]
	if size == nil {
		return
	}

	// 添加额外的外部空间，特别是对于叶子节点
	extraSpace := 5.0
	if len(node.Children) == 0 {
		extraSpace = 15.0 // 叶子节点需要更多空间
	}

	left := node.X - size.Width/2 - extraSpace
	right := node.X + size.Width/2 + extraSpace
	top := node.Y - size.Height/2 - extraSpace
	bottom := node.Y + size.Height/2 + extraSpace

	bounds.MinX = math.Min(bounds.MinX, left)
	bounds.MaxX = math.Max(bounds.MaxX, right)
	bounds.MinY = math.Min(bounds.MinY, top)
	bounds.MaxY = math.Max(bounds.MaxY, bottom)

	for _, child := range node.Children {
		calculateBoundsWithSizes(child, nodeSizes, bounds)
	}
}

func getNodeStyle(node *types.Node, isRoot bool, config *DrawConfig) *types.NodeStyle {
	if node.Style != nil {
		return node.Style
	}

	// 如果有主题配置，使用主题的样式
	if config.Theme != nil {
		nodeStyles := config.Theme.GetNodeStyles()

		if isRoot {
			return nodeStyles["root"]
		}

		// 检查是否为叶子节点
		if len(node.Children) == 0 {
			return nodeStyles["leaf"]
		}

		// 根据子节点类型判断层级
		hasGrandchildren := false
		for _, child := range node.Children {
			if len(child.Children) > 0 {
				hasGrandchildren = true
				break
			}
		}

		if hasGrandchildren {
			return nodeStyles["level1"]
		} else {
			return nodeStyles["level2"]
		}
	}

	// 如果没有主题配置，使用默认样式
	if isRoot {
		return &types.NodeStyle{
			FillColor:   [3]float64{0.051, 0.043, 0.133},
			StrokeColor: [3]float64{0.051, 0.043, 0.133},
			TextColor:   [3]float64{1.0, 1.0, 1.0},
		}
	}

	// 检查是否为叶子节点
	if len(node.Children) == 0 {
		return &types.NodeStyle{
			FillColor:   [3]float64{1.0, 1.0, 1.0},
			StrokeColor: [3]float64{1.0, 1.0, 1.0},
			TextColor:   [3]float64{0.0, 0.0, 0.0},
		}
	}

	return &types.NodeStyle{
		FillColor:   [3]float64{0.96, 0.97, 0.98},
		StrokeColor: [3]float64{0.96, 0.97, 0.98},
		TextColor:   [3]float64{0.0, 0.0, 0.0},
	}
}

func drawRoundedRect(dc *gg.Context, x, y, w, h, r float64) {
	// Ensure radius is not too large
	r = math.Min(r, math.Min(w/2, h/2))

	// Start path
	dc.NewSubPath()

	// Top edge and top-right corner
	dc.MoveTo(x+r, y)
	dc.LineTo(x+w-r, y)
	dc.DrawArc(x+w-r, y+r, r, -math.Pi/2, 0)

	// Right edge and bottom-right corner
	dc.LineTo(x+w, y+h-r)
	dc.DrawArc(x+w-r, y+h-r, r, 0, math.Pi/2)

	// Bottom edge and bottom-left corner
	dc.LineTo(x+r, y+h)
	dc.DrawArc(x+r, y+h-r, r, math.Pi/2, math.Pi)

	// Left edge and top-left corner
	dc.LineTo(x, y+r)
	dc.DrawArc(x+r, y+r, r, math.Pi, math.Pi*1.5)

	// Close the path
	dc.ClosePath()
}
