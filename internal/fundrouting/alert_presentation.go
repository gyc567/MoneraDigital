package fundrouting

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"
)

func routingAlertPresentation(deliveries []claimedDelivery) (string, string, map[string]string) {
	if len(deliveries) == 0 {
		return "WARN", "Safeheron 路由告警", map[string]string{}
	}
	payloads := make([]map[string]any, 0, len(deliveries))
	severity := deliveries[0].Severity
	allSLA := true
	allRecovery := true
	for _, delivery := range deliveries {
		if routingSeverityRank(delivery.Severity) > routingSeverityRank(severity) {
			severity = delivery.Severity
		}
		if delivery.AlertType != "SLA_ESCALATION" {
			allSLA = false
		}
		if delivery.AlertType != "RECOVERY_SUMMARY" {
			allRecovery = false
		}
		payloads = append(payloads, decodeRoutingAlertPayload(delivery.Payload))
	}
	if !allSLA && !allRecovery {
		fields := make(map[string]string)
		for key, value := range payloads[0] {
			fields[key] = routingAlertValue(value)
		}
		return severity, "Safeheron routing " + deliveries[0].AlertType, fields
	}

	direction := routingBatchDirection(payloads)
	titleDirection := "交易"
	switch direction {
	case "OUTFLOW":
		titleDirection = "出账"
	case "INFLOW":
		titleDirection = "入账"
	case "INTERNAL_TRANSFER":
		titleDirection = "内部划转"
	}
	fields := map[string]string{"交易数量": strconv.Itoa(len(payloads))}
	if allRecovery {
		fields["恢复说明"] = "此前超时的 Safeheron 交易已进入终态，系统已按最终状态进入后续幂等处理流程。"
	} else {
		fields["告警原因"] = "Safeheron 交易超过约定时限后仍尚未进入终态，系统已执行单笔 API 状态核验，结果见交易明细。"
		fields["处理建议"] = "请在 Safeheron 控制台核对待审批、待签名或链上广播状态；API 核验失败时先检查 API 凭据和网络。"
	}
	if environment := routingAlertString(payloads[0], "environment"); environment != "" {
		fields["环境"] = environment
	}
	for index, payload := range payloads {
		detail := routingSLAAlertDetail(payload)
		if allRecovery {
			detail = routingRecoveryAlertDetail(payload)
		}
		fields[fmt.Sprintf("交易%02d", index+1)] = detail
	}
	titleState := "停留超时"
	if allRecovery {
		titleState = "状态已收敛"
	}
	return severity, fmt.Sprintf("Safeheron %s%s（%d笔）", titleDirection, titleState, len(payloads)), fields
}

func routingRecoveryAlertDetail(payload map[string]any) string {
	detail := routingSLAAlertDetail(payload)
	additional := make([]string, 0, 2)
	if decision := routingAlertString(payload, "resolved_decision"); decision != "" {
		additional = append(additional, "处理结果："+decision)
	}
	if reason := routingAlertString(payload, "resolved_reason_code"); reason != "" {
		additional = append(additional, "处理原因："+reason)
	}
	if detail == "" {
		return strings.Join(additional, "\n")
	}
	if len(additional) == 0 {
		return detail
	}
	return detail + "\n" + strings.Join(additional, "\n")
}

func decodeRoutingAlertPayload(raw []byte) map[string]any {
	result := map[string]any{}
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.UseNumber()
	if err := decoder.Decode(&result); err != nil {
		return map[string]any{"payload_error": "告警数据无法解析"}
	}
	return result
}

func routingSLAAlertDetail(payload map[string]any) string {
	lines := make([]string, 0, 18)
	appendLine := func(label, value string) {
		if strings.TrimSpace(value) != "" {
			lines = append(lines, label+"："+value)
		}
	}
	appendLine("Case ID", routingAlertString(payload, "case_id"))
	appendLine("方向", routingDirectionLabel(routingAlertString(payload, "direction")))
	appendLine("类型", routingAlertString(payload, "movement_kind"))
	appendLine("资产", routingAlertString(payload, "raw_coin_key"))
	appendLine("网络", routingAlertString(payload, "network_family"))
	appendLine("金额", routingAlertString(payload, "amount"))
	status := routingAlertString(payload, "transaction_status")
	if status != "" {
		appendLine("状态", status+routingStatusExplanation(status))
	}
	appendLine("子状态", routingAlertString(payload, "transaction_sub_status"))
	appendLine("TxKey", routingAlertString(payload, "safeheron_tx_key"))
	txHash := routingAlertString(payload, "tx_hash")
	if txHash == "" {
		txHash = "尚未广播"
	}
	appendLine("Tx Hash", txHash)
	appendLine("来源地址", routingAlertString(payload, "source_address"))
	appendLine("目标地址", routingAlertString(payload, "destination_address"))
	appendLine("发生时间", routingAlertSGTTime(routingAlertString(payload, "effective_event_time")))
	appendLine("停留时长", routingAlertDuration(routingAlertString(payload, "stuck_seconds")))
	appendLine("最后事件", routingAlertString(payload, "last_source_event_type"))
	appendLine("最后事件时间", routingAlertSGTTime(routingAlertString(payload, "last_source_received_at")))
	appendLine("最后 API 核验", routingAlertSGTTime(routingAlertString(payload, "last_api_checked_at")))
	appendLine("API 核验结果", routingAlertCheckOutcome(payload))
	return strings.Join(lines, "\n")
}

func routingAlertCheckOutcome(payload map[string]any) string {
	outcome := routingAlertString(payload, "last_api_check_outcome")
	if outcome == "ERROR" {
		if code := routingAlertString(payload, "last_api_error_code"); code != "" {
			return "失败（" + code + "）"
		}
		return "失败"
	}
	if outcome == "OBSERVED" {
		return "已核验"
	}
	return outcome
}

func routingAlertString(payload map[string]any, key string) string {
	return routingAlertValue(payload[key])
}

func routingAlertValue(value any) string {
	switch typed := value.(type) {
	case nil:
		return ""
	case string:
		return typed
	case json.Number:
		return typed.String()
	case float64:
		return strconv.FormatFloat(typed, 'f', -1, 64)
	default:
		return fmt.Sprint(typed)
	}
}

func routingBatchDirection(payloads []map[string]any) string {
	direction := ""
	for _, payload := range payloads {
		current := strings.ToUpper(strings.TrimSpace(routingAlertString(payload, "direction")))
		if direction == "" {
			direction = current
			continue
		}
		if current != direction {
			return ""
		}
	}
	return direction
}

func routingDirectionLabel(value string) string {
	switch strings.ToUpper(strings.TrimSpace(value)) {
	case "OUTFLOW":
		return "出账"
	case "INFLOW":
		return "入账"
	case "INTERNAL_TRANSFER":
		return "内部划转"
	default:
		return value
	}
}

func routingStatusExplanation(value string) string {
	switch strings.ToUpper(strings.TrimSpace(value)) {
	case "SUBMITTED":
		return "（已提交，尚未进入终态）"
	case "SIGNING":
		return "（审批已通过或处理中，正在签名）"
	case "BROADCASTING":
		return "（正在广播到区块链）"
	case "CONFIRMING":
		return "（已上链，等待区块确认）"
	case "COMPLETED":
		return "（已完成）"
	case "FAILED":
		return "（失败）"
	case "CANCELLED":
		return "（已取消）"
	case "REJECTED":
		return "（已拒绝）"
	default:
		return ""
	}
}

func routingAlertSGTTime(value string) string {
	if strings.TrimSpace(value) == "" {
		return ""
	}
	parsed, err := time.Parse(time.RFC3339Nano, value)
	if err != nil {
		return value
	}
	location, err := time.LoadLocation("Asia/Singapore")
	if err != nil {
		location = time.FixedZone("UTC+8", 8*60*60)
	}
	return parsed.In(location).Format("2006-01-02 15:04:05 UTC+8")
}

func routingAlertDuration(value string) string {
	seconds, err := strconv.ParseInt(strings.TrimSpace(value), 10, 64)
	if err != nil || seconds < 0 {
		return value
	}
	hours := seconds / 3600
	minutes := (seconds % 3600) / 60
	switch {
	case hours > 0 && minutes > 0:
		return fmt.Sprintf("%d小时%d分钟", hours, minutes)
	case hours > 0:
		return fmt.Sprintf("%d小时", hours)
	default:
		return fmt.Sprintf("%d分钟", minutes)
	}
}

func routingSeverityRank(value string) int {
	switch strings.ToUpper(strings.TrimSpace(value)) {
	case "CRITICAL":
		return 4
	case "ERROR":
		return 3
	case "WARN":
		return 2
	case "INFO":
		return 1
	default:
		return 0
	}
}
