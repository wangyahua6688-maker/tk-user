package repo

import (
	"context"
	"errors"
	"strconv"
	"strings"

	"gorm.io/gorm"
	"tk-common/models"
)

// resolveForumDrawPayload 根据帖子关联关系解析“详情页顶部开奖块”。
func (r *Repository) resolveForumDrawPayload(ctx context.Context, row topicRow) (map[string]interface{}, error) {
	// 1) 优先按 lottery_info_id 直接命中开奖记录（兼容把 draw_record_id 存在该字段的场景）。
	if row.LotteryInfoID > 0 {
		// 定义并初始化当前变量。
		record := models.WDrawRecord{}
		// 定义并初始化当前变量。
		err := r.db.WithContext(ctx).
			// 更新当前变量或字段值。
			Where("id = ? AND status = 1", row.LotteryInfoID).
			// 调用First完成当前处理。
			First(&record).Error
		// 判断条件并进入对应分支逻辑。
		if err == nil {
			// 返回当前处理结果。
			return r.buildForumDrawFromRecord(record), nil
		}
		// 判断条件并进入对应分支逻辑。
		if err != nil && !isNotFound(err) {
			// 返回当前处理结果。
			return nil, err
		}
	}

	// 2) 按图纸记录反查同期开奖（帖子当前常用关联方式）。
	if row.LotteryInfoID > 0 {
		// 定义并初始化当前变量。
		info := models.WLotteryInfo{}
		// 定义并初始化当前变量。
		err := r.db.WithContext(ctx).
			// 更新当前变量或字段值。
			Where("id = ? AND status = 1", row.LotteryInfoID).
			// 调用First完成当前处理。
			First(&info).Error
		// 判断条件并进入对应分支逻辑。
		if err == nil {
			// 定义并初始化当前变量。
			record := models.WDrawRecord{}
			// 定义并初始化当前变量。
			drawErr := r.db.WithContext(ctx).
				// 更新当前变量或字段值。
				Where("special_lottery_id = ? AND issue = ? AND status = 1", info.SpecialLotteryID, info.Issue).
				// 调用Order完成当前处理。
				Order("draw_at DESC, id DESC").
				// 调用First完成当前处理。
				First(&record).Error
			// 判断条件并进入对应分支逻辑。
			if drawErr == nil {
				// 返回当前处理结果。
				return r.buildForumDrawFromRecord(record), nil
			}
			// 判断条件并进入对应分支逻辑。
			if drawErr != nil && !isNotFound(drawErr) {
				// 返回当前处理结果。
				return nil, drawErr
			}
			// 开奖记录不存在时，回退到图纸内置开奖字段。
			return r.buildForumDrawFromInfo(info), nil
		}
		// 判断条件并进入对应分支逻辑。
		if err != nil && !isNotFound(err) {
			// 返回当前处理结果。
			return nil, err
		}
	}

	// 3) 兜底：若列表已带 issue/special_lottery_id，直接按期号反查开奖记录。
	if row.SpecialLotteryID > 0 && strings.TrimSpace(row.Issue) != "" {
		// 定义并初始化当前变量。
		record := models.WDrawRecord{}
		// 定义并初始化当前变量。
		err := r.db.WithContext(ctx).
			// 更新当前变量或字段值。
			Where("special_lottery_id = ? AND issue = ? AND status = 1", row.SpecialLotteryID, row.Issue).
			// 调用Order完成当前处理。
			Order("draw_at DESC, id DESC").
			// 调用First完成当前处理。
			First(&record).Error
		// 判断条件并进入对应分支逻辑。
		if err == nil {
			// 返回当前处理结果。
			return r.buildForumDrawFromRecord(record), nil
		}
		// 判断条件并进入对应分支逻辑。
		if err != nil && !isNotFound(err) {
			// 返回当前处理结果。
			return nil, err
		}
	}

	// 4) 未关联开奖时返回空对象，前端按空状态展示。
	return nil, nil
}

// buildForumDrawFromRecord 将开奖记录转换为论坛详情顶部开奖结构。
func (r *Repository) buildForumDrawFromRecord(record models.WDrawRecord) map[string]interface{} {
	// 1) 解析 6+1 号码。
	numbers := extractDrawNumbersFromRecord(record)
	// 2) 提取“生肖/五行”组合标签与独立标签。
	labels := extractDrawLabels(record, numbers)
	// 定义并初始化当前变量。
	zodiac, wuxing := extractZodiacAndWuxingLabels(record, numbers)
	// 3) 返回统一结构。
	return map[string]interface{}{
		// 处理当前语句逻辑。
		"id": record.ID,
		// 处理当前语句逻辑。
		"special_lottery_id": record.SpecialLotteryID,
		// 处理当前语句逻辑。
		"issue": record.Issue,
		// 处理当前语句逻辑。
		"year": record.Year,
		// 处理当前语句逻辑。
		"draw_at": record.DrawAt,
		// 处理当前语句逻辑。
		"numbers": numbers,
		// 处理当前语句逻辑。
		"labels": labels,
		// 处理当前语句逻辑。
		"zodiac_labels": zodiac,
		// 处理当前语句逻辑。
		"wuxing_labels": wuxing,
		// 处理当前语句逻辑。
		"playback_url": record.PlaybackURL,
	}
}

// buildForumDrawFromInfo 将图库图纸中的开奖字段兜底转换为开奖块。
func (r *Repository) buildForumDrawFromInfo(info models.WLotteryInfo) map[string]interface{} {
	// 1) 解析图纸内置开奖串。
	numbers := extractDrawNumbersFromInfo(info)
	// 2) 图纸无独立标签字段时，用占位“生肖/五行”规则生成。
	labels := buildPairLabels(numbers)
	// 定义并初始化当前变量。
	zodiac := make([]string, 0, len(labels))
	// 定义并初始化当前变量。
	wuxing := make([]string, 0, len(labels))
	// 循环处理当前数据集合。
	for _, item := range labels {
		// 定义并初始化当前变量。
		z, w := splitPairLabel(item)
		// 更新当前变量或字段值。
		zodiac = append(zodiac, z)
		// 更新当前变量或字段值。
		wuxing = append(wuxing, w)
	}
	// 3) 返回统一结构（id 采用图纸 id，便于前端仍可渲染）。
	return map[string]interface{}{
		// 处理当前语句逻辑。
		"id": info.ID,
		// 处理当前语句逻辑。
		"special_lottery_id": info.SpecialLotteryID,
		// 处理当前语句逻辑。
		"issue": info.Issue,
		// 处理当前语句逻辑。
		"year": info.Year,
		// 处理当前语句逻辑。
		"draw_at": info.DrawAt,
		// 处理当前语句逻辑。
		"numbers": numbers,
		// 处理当前语句逻辑。
		"labels": labels,
		// 处理当前语句逻辑。
		"zodiac_labels": zodiac,
		// 处理当前语句逻辑。
		"wuxing_labels": wuxing,
		// 处理当前语句逻辑。
		"playback_url": info.PlaybackURL,
	}
}

// isNotFound 判断是否为 gorm 记录不存在错误。
func isNotFound(err error) bool {
	// 返回当前处理结果。
	return errors.Is(err, gorm.ErrRecordNotFound)
}

// splitCSVInts 将逗号分隔号码串解析为整型切片。
func splitCSVInts(raw string) []int {
	// 1) 多分隔符统一切分。
	parts := strings.FieldsFunc(strings.TrimSpace(raw), func(r rune) bool {
		// 返回当前处理结果。
		return r == ',' || r == '|' || r == '/' || r == ' ' || r == '\t' || r == '\n'
	})
	// 2) 逐项转整数。
	out := make([]int, 0, len(parts))
	// 循环处理当前数据集合。
	for _, item := range parts {
		// 定义并初始化当前变量。
		n, err := strconv.Atoi(strings.TrimSpace(item))
		// 判断条件并进入对应分支逻辑。
		if err != nil {
			// 处理当前语句逻辑。
			continue
		}
		// 更新当前变量或字段值。
		out = append(out, n)
	}
	// 返回当前处理结果。
	return out
}

// extractDrawNumbersFromRecord 从开奖记录中提取 6+1 号码。
func extractDrawNumbersFromRecord(record models.WDrawRecord) []int {
	// 1) 优先读取普通号 + 特别号字段。
	normal := splitCSVInts(record.NormalDrawResult)
	// 定义并初始化当前变量。
	special := splitCSVInts(record.SpecialDrawResult)
	// 判断条件并进入对应分支逻辑。
	if len(normal) == 6 && len(special) == 1 {
		// 返回当前处理结果。
		return append(normal, special[0])
	}
	// 2) 兼容旧字段 draw_result。
	return splitCSVInts(record.DrawResult)
}

// extractDrawNumbersFromInfo 从图纸表中提取 6+1 号码。
func extractDrawNumbersFromInfo(info models.WLotteryInfo) []int {
	// 1) 优先读取普通号 + 特别号字段。
	normal := splitCSVInts(info.NormalDrawResult)
	// 定义并初始化当前变量。
	special := splitCSVInts(info.SpecialDrawResult)
	// 判断条件并进入对应分支逻辑。
	if len(normal) == 6 && len(special) == 1 {
		// 返回当前处理结果。
		return append(normal, special[0])
	}
	// 2) 兼容旧字段 draw_result。
	return splitCSVInts(info.DrawResult)
}

// extractDrawLabels 提取开奖记录的组合标签（生肖/五行）。
func extractDrawLabels(record models.WDrawRecord, numbers []int) []string {
	// 1) 优先读取 draw_labels。
	labels := splitCSVLabels(record.DrawLabels)
	// 判断条件并进入对应分支逻辑。
	if len(labels) == len(numbers) && len(labels) > 0 {
		// 返回当前处理结果。
		return labels
	}
	// 2) 缺失时自动生成占位标签。
	return buildPairLabels(numbers)
}

// extractZodiacAndWuxingLabels 提取生肖/五行两套标签。
func extractZodiacAndWuxingLabels(record models.WDrawRecord, numbers []int) ([]string, []string) {
	// 1) 优先使用独立字段，避免前端重复拆分。
	zodiac := splitCSVLabels(record.ZodiacLabels)
	// 定义并初始化当前变量。
	wuxing := splitCSVLabels(record.WuxingLabels)
	// 判断条件并进入对应分支逻辑。
	if len(zodiac) == len(numbers) && len(wuxing) == len(numbers) && len(zodiac) > 0 {
		// 返回当前处理结果。
		return zodiac, wuxing
	}

	// 2) 回退由组合标签拆分。
	paired := extractDrawLabels(record, numbers)
	// 更新当前变量或字段值。
	zodiac = make([]string, 0, len(paired))
	// 更新当前变量或字段值。
	wuxing = make([]string, 0, len(paired))
	// 循环处理当前数据集合。
	for _, item := range paired {
		// 定义并初始化当前变量。
		z, w := splitPairLabel(item)
		// 更新当前变量或字段值。
		zodiac = append(zodiac, z)
		// 更新当前变量或字段值。
		wuxing = append(wuxing, w)
	}
	// 返回当前处理结果。
	return zodiac, wuxing
}

// splitCSVLabels 解析标签串（支持逗号、分号、空白符）。
func splitCSVLabels(raw string) []string {
	// 1) 多分隔符切片。
	parts := strings.FieldsFunc(strings.TrimSpace(raw), func(r rune) bool {
		// 返回当前处理结果。
		return r == ',' || r == ';' || r == '|' || r == '\n' || r == '\r' || r == '\t'
	})
	// 2) 清理空字符串。
	out := make([]string, 0, len(parts))
	// 循环处理当前数据集合。
	for _, item := range parts {
		// 定义并初始化当前变量。
		value := strings.TrimSpace(item)
		// 判断条件并进入对应分支逻辑。
		if value == "" {
			// 处理当前语句逻辑。
			continue
		}
		// 更新当前变量或字段值。
		out = append(out, value)
	}
	// 返回当前处理结果。
	return out
}

// buildPairLabels 为号码生成“生肖/五行”占位标签。
func buildPairLabels(numbers []int) []string {
	// 1) 定义生肖与五行序列。
	zodiacs := []string{"鼠", "牛", "虎", "兔", "龙", "蛇", "马", "羊", "猴", "鸡", "狗", "猪"}
	// 定义并初始化当前变量。
	wuxing := []string{"金", "木", "水", "火", "土"}
	// 2) 基于号码取模生成标签。
	out := make([]string, 0, len(numbers))
	// 循环处理当前数据集合。
	for _, n := range numbers {
		// 定义并初始化当前变量。
		zodiac := zodiacs[(n-1+len(zodiacs))%len(zodiacs)]
		// 定义并初始化当前变量。
		element := wuxing[(n-1+len(wuxing))%len(wuxing)]
		// 更新当前变量或字段值。
		out = append(out, zodiac+"/"+element)
	}
	// 返回当前处理结果。
	return out
}

// splitPairLabel 拆分“生肖/五行”格式组合标签。
func splitPairLabel(raw string) (string, string) {
	// 1) 空值直接返回。
	value := strings.TrimSpace(raw)
	// 判断条件并进入对应分支逻辑。
	if value == "" {
		// 返回当前处理结果。
		return "", ""
	}
	// 2) 按首次 "/" 拆分。
	parts := strings.SplitN(value, "/", 2)
	// 判断条件并进入对应分支逻辑。
	if len(parts) == 2 {
		// 返回当前处理结果。
		return strings.TrimSpace(parts[0]), strings.TrimSpace(parts[1])
	}
	// 3) 非组合格式时把原值作为生肖返回。
	return value, ""
}
