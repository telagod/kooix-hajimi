package scanner

import (
	"bufio"
	"fmt"
	"os"
	"regexp"
	"strings"
)

// QueryPhase 查询阶段
type QueryPhase struct {
	Name        string
	Priority    int
	Description string
	Queries     []string
}

// PhasedQueryManager 分阶段查询管理器
type PhasedQueryManager struct {
	phases []QueryPhase
}

// NewPhasedQueryManager 创建新的分阶段查询管理器
func NewPhasedQueryManager() *PhasedQueryManager {
	return &PhasedQueryManager{
		phases: make([]QueryPhase, 0),
	}
}

// LoadPhasedQueries 从文件加载分阶段查询
func (qm *PhasedQueryManager) LoadPhasedQueries(filename string) error {
	file, err := os.Open(filename)
	if err != nil {
		return fmt.Errorf("failed to open query file: %v", err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	var currentPhase *QueryPhase
	
	phaseRegex := regexp.MustCompile(`# \*\*\[PHASE (\d+)\] (.+?) \*\*`)
	
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		
		// 跳过空行和普通注释
		if line == "" || (strings.HasPrefix(line, "#") && !strings.Contains(line, "[PHASE")) {
			continue
		}
		
		// 检查是否是新阶段
		if matches := phaseRegex.FindStringSubmatch(line); len(matches) > 2 {
			// 保存前一个阶段
			if currentPhase != nil {
				qm.phases = append(qm.phases, *currentPhase)
			}
			
			// 创建新阶段
			currentPhase = &QueryPhase{
				Name:        fmt.Sprintf("Phase %s", matches[1]),
				Priority:    len(qm.phases) + 1,
				Description: matches[2],
				Queries:     make([]string, 0),
			}
			continue
		}
		
		// 添加查询到当前阶段
		if currentPhase != nil && !strings.HasPrefix(line, "#") {
			currentPhase.Queries = append(currentPhase.Queries, line)
		}
	}
	
	// 保存最后一个阶段
	if currentPhase != nil {
		qm.phases = append(qm.phases, *currentPhase)
	}
	
	return scanner.Err()
}

// GetPhases 获取所有查询阶段
func (qm *PhasedQueryManager) GetPhases() []QueryPhase {
	return qm.phases
}

// GetPhaseByPriority 按优先级获取阶段
func (qm *PhasedQueryManager) GetPhaseByPriority(priority int) *QueryPhase {
	for _, phase := range qm.phases {
		if phase.Priority == priority {
			return &phase
		}
	}
	return nil
}

// GetQueriesByPhase 获取指定阶段的查询
func (qm *PhasedQueryManager) GetQueriesByPhase(phaseName string) []string {
	for _, phase := range qm.phases {
		if phase.Name == phaseName {
			return phase.Queries
		}
	}
	return nil
}

// GetAllQueries 获取所有查询（保持向后兼容）
func (qm *PhasedQueryManager) GetAllQueries() []string {
	var allQueries []string
	for _, phase := range qm.phases {
		allQueries = append(allQueries, phase.Queries...)
	}
	return allQueries
}

// GetQueryStats 获取查询统计信息
func (qm *PhasedQueryManager) GetQueryStats() map[string]interface{} {
	stats := make(map[string]interface{})
	
	totalQueries := 0
	phaseStats := make(map[string]int)
	
	for _, phase := range qm.phases {
		count := len(phase.Queries)
		totalQueries += count
		phaseStats[phase.Name] = count
	}
	
	stats["total_queries"] = totalQueries
	stats["total_phases"] = len(qm.phases)
	stats["phase_breakdown"] = phaseStats
	
	return stats
}