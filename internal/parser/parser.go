package parser

import (
	"bufio"
	"regexp"
	"strings"

	"github.com/hellodeveye/mindmapgen/pkg/types"
)

func Parse(input string) (*types.Node, error) {
	scanner := bufio.NewScanner(strings.NewReader(input))
	var stack []*types.Node
	var root *types.Node
	foundMindmap := false

	// 检测使用的缩进方式
	indentType := detectIndentationType(input)

	// 记录每个层级的最后一个节点
	levelLastNodes := make(map[int]*types.Node)

	// 记录上一行的缩进级别，用于检测层级变化
	prevLevel := -1

	for scanner.Scan() {
		line := scanner.Text()
		trimmed := strings.TrimSpace(line)

		if trimmed == "" {
			continue
		}

		if trimmed == "mindmap" {
			foundMindmap = true
			continue
		}

		level := getIndentationLevel(line, indentType)

		// 清理文本，对根节点做特殊处理
		cleanedText := cleanText(trimmed)
		if (level == 0 && !foundMindmap) || (level == 1 && foundMindmap) {
			// 根节点特殊处理，移除"root"和双括号
			cleanedText = cleanRootText(cleanedText)
		}

		node := &types.Node{
			Text:     cleanedText,
			Children: []*types.Node{},
		}

		if !foundMindmap && level == 0 {
			root = node
			stack = []*types.Node{node}
			levelLastNodes[level] = node
			prevLevel = level
		} else if foundMindmap && level == 1 { // First node after mindmap is root
			root = node
			stack = []*types.Node{node}
			levelLastNodes[level] = node
			prevLevel = level
			foundMindmap = false // Reset flag
		} else if root != nil {
			// 根据当前缩进级别和上一级别的关系确定父节点
			if level > prevLevel {
				// 当前级别比上一级别深一级，正常添加为子节点
				parent := levelLastNodes[prevLevel]
				if parent != nil {
					parent.Children = append(parent.Children, node)
					// 更新堆栈和层级记录
					if len(stack) > level {
						stack[level] = node
					} else {
						stack = append(stack, node)
					}
					levelLastNodes[level] = node
				}
			} else {
				// 当前级别与上一级别相同或更浅，需要找到正确的父节点
				parentLevel := level - 1
				if parentLevel >= 0 && levelLastNodes[parentLevel] != nil {
					parent := levelLastNodes[parentLevel]
					parent.Children = append(parent.Children, node)

					// 更新堆栈，清除后续层级的记录
					if len(stack) > level {
						stack = stack[:level]
						stack = append(stack, node)

						// 清除更深层级的记录
						for l := level + 1; l <= prevLevel; l++ {
							delete(levelLastNodes, l)
						}
					} else {
						stack = append(stack, node)
					}

					levelLastNodes[level] = node
				}
			}

			prevLevel = level
		}
	}

	if root == nil {
		root = &types.Node{
			Text:     "Root",
			Children: []*types.Node{},
		}
	}

	return root, scanner.Err()
}

// 检测使用的缩进类型
func detectIndentationType(input string) string {
	lines := strings.Split(input, "\n")
	tabCount := 0
	spaceCount := 0

	for _, line := range lines {
		if strings.HasPrefix(line, "\t") {
			tabCount++
		} else if strings.HasPrefix(line, "  ") {
			spaceCount++
		}
	}

	if tabCount > spaceCount {
		return "tab"
	}
	return "space"
}

// 根据缩进类型获取缩进级别
func getIndentationLevel(line string, indentType string) int {
	if indentType == "tab" {
		// 计算开头的制表符数量
		tabCount := 0
		for _, c := range line {
			if c == '\t' {
				tabCount++
			} else {
				break
			}
		}
		return tabCount
	} else {
		// 使用原始的空格计数方法
		return countIndentation(line)
	}
}

func countIndentation(line string) int {
	count := 0
	for _, c := range line {
		if c == ' ' {
			count++
		} else if c == '\t' {
			// 每个tab算作一个层级
			count += 2
		} else {
			break
		}
	}
	return count / 2 // 每两个空格为一个层级，tab已经转换为相应空格数
}

// 清理普通节点文本
func cleanText(text string) string {
	// 删除前缀的空格、制表符和破折号
	text = strings.TrimLeft(text, " \t-")
	return strings.TrimSpace(text)
}

// 专门处理根节点文本，移除"root"和双括号
func cleanRootText(text string) string {
	// 先使用常规清理
	text = cleanText(text)

	// 移除开头的"root"
	text = strings.TrimPrefix(text, "root")

	// 移除双括号
	re := regexp.MustCompile(`^\(\((.*)\)\)$`)
	matches := re.FindStringSubmatch(text)
	if len(matches) > 1 {
		return matches[1]
	}

	return strings.TrimSpace(text)
}
