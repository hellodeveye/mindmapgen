package drawer

import (
	"fmt"
	"math"
	"os"
	"path/filepath"
	"unicode"

	"github.com/fogleman/gg"
	"github.com/hellodeveye/mindmapgen/pkg/types"
)

const (
	MinNodeWidth  = 100.0 // 最小节点宽度
	MaxNodeWidth  = 150.0 // 最大节点宽度
	MinNodeHeight = 28.0  // 最小节点高度
	LevelSpacing  = 120.0 // 水平层级间距
	NodeSpacing   = 20.0  // 垂直节点间距
	CornerRadius  = 5.0
	FontSize      = 11.0
	MarginPercent = 0.10 // 边距比例
	Scale         = 1.0
	LineHeight    = 16.0
	TextPadding   = 10.0 // 文本内边距

	// 力导向布局参数
	RepulsionForce      = 2000.0 // 节点间斥力系数，降低以减少节点间距
	AttractionForce     = 0.8    // 连接线引力系数，增加以缩短连接线
	MaxIterations       = 100    // 最大迭代次数
	CoolingFactor       = 0.95   // 冷却因子，用于减小每次迭代的移动距离
	MinimumEnergy       = 0.01   // 最小能量阈值，低于此值停止迭代
	MaximumDisplacement = 30.0   // 最大移动距离限制
	HierarchyFactor     = 2.0    // 层次结构约束因子，确保思维导图层次清晰
)

var (
	rootStyle = &types.NodeStyle{
		FillColor:   [3]float64{0.94, 0.98, 1.0},
		StrokeColor: [3]float64{0.4, 0.6, 0.8},
		TextColor:   [3]float64{0.2, 0.2, 0.2},
	}
	level1Style = &types.NodeStyle{
		FillColor:   [3]float64{0.96, 0.99, 0.96},
		StrokeColor: [3]float64{0.5, 0.75, 0.5},
		TextColor:   [3]float64{0.2, 0.2, 0.2},
	}
	level2Style = &types.NodeStyle{
		FillColor:   [3]float64{0.99, 0.96, 0.94},
		StrokeColor: [3]float64{0.75, 0.6, 0.4},
		TextColor:   [3]float64{0.2, 0.2, 0.2},
	}
	leafStyle = &types.NodeStyle{
		FillColor:   [3]float64{0.98, 0.98, 0.98},
		StrokeColor: [3]float64{0.7, 0.7, 0.7},
		TextColor:   [3]float64{0.2, 0.2, 0.2},
	}
)

type Bounds struct {
	MinX, MinY, MaxX, MaxY float64
}

// 存储节点尺寸的结构
type NodeSize struct {
	Width  float64
	Height float64
	Lines  []string // 存储换行后的文本
}

func loadFont(dc *gg.Context) error {
	// Get current working directory
	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get working directory: %v", err)
	}

	// Try to find the project root (where assets directory is)
	projectRoot := cwd
	for {
		if _, err := os.Stat(filepath.Join(projectRoot, "assets")); err == nil {
			break
		}
		parent := filepath.Dir(projectRoot)
		if parent == projectRoot {
			return fmt.Errorf("could not find project root directory")
		}
		projectRoot = parent
	}

	// Calculate font size based on context resolution
	size := FontSize * Scale

	// Try to load fonts from assets directory
	fonts := []string{
		filepath.Join(projectRoot, "assets", "fonts", "SourceHanSansSC-Regular.otf"),
		filepath.Join(projectRoot, "assets", "fonts", "NotoSansSC-Regular.ttf"),
		filepath.Join(projectRoot, "assets", "fonts", "simhei.ttf"),
	}

	for _, fontPath := range fonts {
		if err := dc.LoadFontFace(fontPath, size); err == nil {
			return nil
		}
	}

	// If all attempts failed, use default font
	dc.LoadFontFace("", size)
	return fmt.Errorf("failed to load preferred fonts, using default font")
}

func calculateBounds(node *types.Node, x, y float64, bounds *Bounds, nodeSizes map[*types.Node]*NodeSize) {
	if node == nil {
		return
	}

	// Get node size from the map
	size := nodeSizes[node]
	if size == nil {
		return
	}

	// Update bounds with current node
	left := x - size.Width/2
	right := x + size.Width/2
	top := y - size.Height/2
	bottom := y + size.Height/2

	bounds.MinX = math.Min(bounds.MinX, left)
	bounds.MaxX = math.Max(bounds.MaxX, right)
	bounds.MinY = math.Min(bounds.MinY, top)
	bounds.MaxY = math.Max(bounds.MaxY, bottom)

	// Recursively calculate bounds for children
	for _, child := range node.Children {
		calculateBounds(child, child.X, child.Y, bounds, nodeSizes)
	}
}

// 保存对根节点的引用，用于识别根节点
var root *types.Node

func Draw(rootNode *types.Node, filename string) error {
	// 创建临时上下文用于文本测量
	tempDC := gg.NewContext(1, 1)
	if err := loadFont(tempDC); err != nil {
		fmt.Printf("Warning: %v\n", err)
	}

	// 计算节点尺寸
	nodeSizes := make(map[*types.Node]*NodeSize)
	calculateNodeSizes(tempDC, rootNode, nodeSizes)

	// 获取树的深度和每层节点数（可能不再需要，但保留）
	maxDepth := 0
	levelCounts := make(map[int]int)
	calculateTreeMetrics(rootNode, 0, &maxDepth, levelCounts)

	// 保存根节点引用
	root = rootNode

	// 计算水平思维导图布局
	subtreeHeights := make(map[*types.Node]float64)
	calculateSubtreeHeights(rootNode, nodeSizes, subtreeHeights)
	horizontalMindmapLayout(rootNode, 0, 0, nodeSizes, subtreeHeights)

	// 计算边界
	bounds := &Bounds{
		MinX: math.MaxFloat64,
		MinY: math.MaxFloat64,
		MaxX: -math.MaxFloat64,
		MaxY: -math.MaxFloat64,
	}
	calculateBoundsWithSizes(rootNode, nodeSizes, bounds)

	// 扩展边界，确保有足够的边距
	extraMargin := 50.0 // 增加固定边距
	bounds.MinX -= extraMargin
	bounds.MinY -= extraMargin
	bounds.MaxX += extraMargin
	bounds.MaxY += extraMargin

	// 计算画布尺寸
	contentWidth := bounds.MaxX - bounds.MinX
	contentHeight := bounds.MaxY - bounds.MinY

	// 使用固定边距，确保左侧有足够空间
	canvasWidth := contentWidth
	canvasHeight := contentHeight

	// 创建最终上下文
	dc := gg.NewContext(int(canvasWidth*Scale), int(canvasHeight*Scale))
	dc.SetLineWidth(1.0 * Scale) // 线条稍微细一点
	dc.SetLineJoin(gg.LineJoinRound)
	dc.SetLineCap(gg.LineCapButt) // 直角连接线使用Butt

	if err := loadFont(dc); err != nil {
		fmt.Printf("Warning: %v\n", err)
	}

	// 设置背景
	dc.SetRGB(1, 1, 1)
	dc.Clear()

	// 绘制温和的网格背景
	drawGridBackground(dc, canvasWidth, canvasHeight)

	// 应用变换 - 将图形的原点移动到 (0, 0) 处
	dc.Scale(Scale, Scale)
	dc.Translate(-bounds.MinX, -bounds.MinY)

	// 先绘制所有连接线
	drawConnectionsHorizontal(dc, rootNode, nodeSizes)

	// 然后绘制所有节点
	drawAllNodes(dc, rootNode, nodeSizes)

	return dc.SavePNG(filename)
}

// 绘制温和的网格背景
func drawGridBackground(dc *gg.Context, width, height float64) {
	// 设置网格线颜色（非常淡的灰色）
	dc.SetRGBA(0, 0, 0, 0.02)

	// 绘制大网格
	gridSize := 40.0 * Scale
	for x := 0.0; x < width; x += gridSize {
		dc.DrawLine(x, 0, x, height)
	}
	for y := 0.0; y < height; y += gridSize {
		dc.DrawLine(0, y, width, y)
	}
	dc.Stroke()
}

// 计算每个节点及其子树所需的总垂直高度
func calculateSubtreeHeights(node *types.Node, nodeSizes map[*types.Node]*NodeSize, subtreeHeights map[*types.Node]float64) {
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
		calculateSubtreeHeights(child, nodeSizes, subtreeHeights)
		totalChildrenHeight += subtreeHeights[child]
	}

	// 加上节点间的垂直间距
	totalChildrenHeight += NodeSpacing * float64(len(node.Children)-1)

	// 子树高度是自身高度和子节点总高度中的较大值
	subtreeHeights[node] = math.Max(nodeSize.Height, totalChildrenHeight)
}

// 水平思维导图布局算法
func horizontalMindmapLayout(node *types.Node, x, y float64, nodeSizes map[*types.Node]*NodeSize, subtreeHeights map[*types.Node]float64) {
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
	childrenTotalHeight += NodeSpacing * float64(len(node.Children)-1)

	currentY := y - childrenTotalHeight/2

	// 子节点的水平位置
	childX := x + nodeSize.Width + LevelSpacing

	// 递归放置子节点
	for _, child := range node.Children {
		childSubtreeHeight := subtreeHeights[child]
		// 将子节点垂直居中在其子树所占空间内
		childY := currentY + childSubtreeHeight/2

		horizontalMindmapLayout(child, childX, childY, nodeSizes, subtreeHeights)

		// 更新下一个子节点的起始Y坐标
		currentY += childSubtreeHeight + NodeSpacing
	}
}

// 绘制水平布局的连接线
func drawConnectionsHorizontal(dc *gg.Context, node *types.Node, nodeSizes map[*types.Node]*NodeSize) {
	if node == nil || len(node.Children) == 0 {
		return
	}

	parentStyle := getNodeStyle(node, node == root)
	parentSize := nodeSizes[node]
	if parentSize == nil {
		return
	}

	// 连接起点（父节点右侧中心）
	startX := node.X + parentSize.Width/2
	startY := node.Y

	for _, child := range node.Children {
		childStyle := getNodeStyle(child, false)
		childSize := nodeSizes[child]
		if childSize == nil {
			continue
		}

		// 连接终点（子节点左侧中心）
		endX := child.X - childSize.Width/2
		endY := child.Y

		// 计算混合颜色
		blendedColor := blendColors(parentStyle.StrokeColor, childStyle.StrokeColor)
		dc.SetRGB(blendedColor[0], blendedColor[1], blendedColor[2])
		dc.SetLineWidth(1.0) // 使用较细的线条

		// 绘制直角连接线
		halfX := startX + (endX-startX)/2
		dc.MoveTo(startX, startY)
		dc.LineTo(halfX, startY) // 水平线
		dc.LineTo(halfX, endY)   // 垂直线
		dc.LineTo(endX, endY)    // 水平线
		dc.Stroke()

		// 递归绘制子节点的连接线
		drawConnectionsHorizontal(dc, child, nodeSizes)
	}
}

// 颜色混合函数
func blendColors(c1, c2 [3]float64) [3]float64 {
	return [3]float64{
		(c1[0] + c2[0]) / 2,
		(c1[1] + c2[1]) / 2,
		(c1[2] + c2[2]) / 2,
	}
}

// 绘制单个节点
func drawSingleNode(dc *gg.Context, node *types.Node, isRoot bool, nodeSizes map[*types.Node]*NodeSize) {
	if node == nil {
		return
	}

	style := getNodeStyle(node, isRoot)
	nodeSize := nodeSizes[node]

	if nodeSize == nil {
		return
	}

	// 计算节点位置
	x := node.X - nodeSize.Width/2
	y := node.Y - nodeSize.Height/2
	w := nodeSize.Width
	h := nodeSize.Height
	r := CornerRadius

	// 绘制节点阴影
	shadowOffset := 2.0
	shadowBlur := 3.0
	for i := 0.0; i < shadowBlur; i += 0.5 {
		opacity := 0.04 * (shadowBlur - i) / shadowBlur
		dc.SetRGBA(0, 0, 0, opacity)
		so := i + shadowOffset
		drawRoundedRect(dc, x+so, y+so, w, h, r)
		dc.Fill()
	}

	// 绘制节点背景
	dc.SetRGB(style.FillColor[0], style.FillColor[1], style.FillColor[2])
	drawRoundedRect(dc, x, y, w, h, r)
	dc.Fill()

	// 绘制节点边框
	dc.SetRGB(style.StrokeColor[0], style.StrokeColor[1], style.StrokeColor[2])
	dc.SetLineWidth(0.8)
	drawRoundedRect(dc, x, y, w, h, r)
	dc.Stroke()

	// 绘制文本
	dc.SetRGB(style.TextColor[0], style.TextColor[1], style.TextColor[2])
	startY := node.Y - (float64(len(nodeSize.Lines))*LineHeight)/2 + LineHeight/2

	for i, line := range nodeSize.Lines {
		y := startY + float64(i)*LineHeight
		dc.DrawStringAnchored(line, node.X, y, 0.5, 0.5)
	}
}

func calculateNodeSizes(dc *gg.Context, node *types.Node, nodeSizes map[*types.Node]*NodeSize) float64 {
	if node == nil {
		return 0
	}

	// 计算文本换行和节点尺寸
	size := calculateTextWrapping(dc, node.Text)
	nodeSizes[node] = size

	// 如果没有子节点，直接返回节点宽度
	if len(node.Children) == 0 {
		return size.Width
	}

	// 递归计算所有子节点的宽度
	totalChildrenWidth := 0.0

	for _, child := range node.Children {
		childWidth := calculateNodeSizes(dc, child, nodeSizes)
		totalChildrenWidth += childWidth
	}

	// 计算最小间距
	minSpacing := 12.0
	spacing := NodeSpacing
	childCount := len(node.Children)

	// 动态调整间距，子节点多时使用更紧凑的布局
	if childCount > 4 {
		reductionFactor := math.Max(0.7, 1.0-float64(childCount-4)*0.03)
		spacing *= reductionFactor
	}
	spacing = math.Max(spacing, minSpacing)

	// 子节点总宽度+间距
	totalChildrenWidthWithSpacing := totalChildrenWidth + spacing*float64(childCount-1)

	// 如果子节点总宽度超过了最大节点宽度的2倍，不再将父节点扩展到子节点宽度
	// 这样可以避免上层节点过宽的问题
	if totalChildrenWidthWithSpacing > MaxNodeWidth*2 {
		return size.Width
	}

	// 父节点宽度 = max(自身内容宽度, 子节点总宽度+间距)
	// 但不超过MaxNodeWidth的1.5倍，避免节点过宽
	size.Width = math.Min(
		math.Max(size.Width, totalChildrenWidthWithSpacing),
		MaxNodeWidth*1.5)

	return size.Width
}

// 修改计算文本换行和节点尺寸的函数，提高效率和美观度
func calculateTextWrapping(dc *gg.Context, text string) *NodeSize {
	words := splitIntoWords(text)
	if len(words) == 0 {
		return &NodeSize{Width: MinNodeWidth, Height: MinNodeHeight}
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
	nodeWidth := textWidth + 2*TextPadding

	// 确保节点宽度在限制范围内
	if nodeWidth < MinNodeWidth {
		nodeWidth = MinNodeWidth
	} else if nodeWidth > MaxNodeWidth {
		nodeWidth = MaxNodeWidth
	}

	// 尝试更紧凑的换行策略
	availableWidth := nodeWidth - 2*TextPadding
	var lines []string

	// 根据文本长度决定换行策略
	if textWidth > MaxNodeWidth*1.5 {
		// 对于长文本，尝试更激进的换行
		lines = breakTextIntoLines(dc, words, availableWidth*0.85)
	} else if textWidth > MaxNodeWidth {
		// 对于中等长度文本，适度换行
		lines = breakTextIntoLines(dc, words, availableWidth*0.9)
	} else {
		// 对于短文本，正常换行
		lines = breakTextIntoLines(dc, words, availableWidth)
	}

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

	// 计算节点高度
	nodeHeight := float64(len(finalLines))*LineHeight + 2*TextPadding
	if nodeHeight < MinNodeHeight {
		nodeHeight = MinNodeHeight
	}

	return &NodeSize{
		Width:  nodeWidth,
		Height: nodeHeight,
		Lines:  finalLines,
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
func drawAllNodes(dc *gg.Context, node *types.Node, nodeSizes map[*types.Node]*NodeSize) {
	if node == nil {
		return
	}

	// 绘制当前节点
	drawSingleNode(dc, node, node == root, nodeSizes)

	// 递归处理所有子节点
	for _, child := range node.Children {
		drawAllNodes(dc, child, nodeSizes)
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

func getNodeStyle(node *types.Node, isRoot bool) *types.NodeStyle {
	if node.Style != nil {
		return node.Style
	}

	// 找出节点的层级
	if isRoot {
		return rootStyle
	}

	// 检查是否为叶子节点
	if len(node.Children) == 0 {
		return leafStyle
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
		return level1Style
	} else {
		return level2Style
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
