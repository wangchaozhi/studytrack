#!/usr/bin/env bash
set -euo pipefail

api_base="${API_BASE:-http://localhost:8080/api/v1}"
web_base="${WEB_BASE:-http://localhost:3000}"
today="$(date +%F)"
exam_date="$(date -d '+180 days' +%F)"
week_start="$(date -d "$(date +%F) -$((($(date +%u)-1))) days" +%F)"
email="ci-$(date +%s)-${RANDOM}@example.test"

curl -fsS "${api_base%/api/v1}/healthz" >/dev/null
curl -fsS "$web_base" | grep -q 'StudyTrack'

register="$(curl -fsS -X POST "$api_base/auth/register" -H 'Content-Type: application/json' -d "{\"email\":\"$email\",\"password\":\"Acceptance123!\",\"nickname\":\"CI验收\",\"timezone\":\"Asia/Shanghai\"}")"
token="$(jq -er '.token' <<<"$register")"
auth=(-H "Authorization: Bearer $token" -H 'Content-Type: application/json')

goal="$(curl -fsS -X POST "$api_base/goals" "${auth[@]}" -d "{\"name\":\"考研验收\",\"type\":\"postgraduate\",\"start_date\":\"$today\",\"exam_date\":\"$exam_date\",\"status\":\"active\",\"target_score\":390,\"settings\":{\"school\":\"XX大学\",\"major\":\"计算机\",\"politics_target\":70,\"english_target\":75,\"math_target\":120,\"professional_target\":125}}")"
goal_id="$(jq -er '.id' <<<"$goal")"
phases="$(curl -fsS "$api_base/goals/$goal_id/phases" "${auth[@]}")"
subjects="$(curl -fsS "$api_base/goals/$goal_id/subjects" "${auth[@]}")"
jq -e 'length == 4' <<<"$phases" >/dev/null
jq -e 'length == 4 and any(.[]; .name == "数学" and .target_score == 120)' <<<"$subjects" >/dev/null
phase_id="$(jq -er '.[0].id' <<<"$phases")"
math_id="$(jq -er '.[] | select(.name == "数学") | .id' <<<"$subjects")"

knowledge="$(curl -fsS -X POST "$api_base/knowledge-points" "${auth[@]}" -d "{\"subject_id\":\"$math_id\",\"name\":\"概率论\",\"importance\":\"core\",\"mastery\":42,\"review_status\":\"review\",\"wrong_count\":2}")"
knowledge_id="$(jq -er '.id' <<<"$knowledge")"
jq -e '.mastery == 42 and .review_status == "review"' <<<"$knowledge" >/dev/null

task="$(curl -fsS -X POST "$api_base/tasks" "${auth[@]}" -d "{\"goal_id\":\"$goal_id\",\"phase_id\":\"$phase_id\",\"subject_id\":\"$math_id\",\"knowledge_point_id\":\"$knowledge_id\",\"title\":\"高数极限练习\",\"task_type\":\"practice\",\"priority\":\"high\",\"planned_date\":\"$today\",\"estimated_minutes\":45}")"
task_id="$(jq -er '.id' <<<"$task")"
session="$(curl -fsS -X POST "$api_base/study-sessions/start" "${auth[@]}" -d "{\"goal_id\":\"$goal_id\",\"task_id\":\"$task_id\",\"subject_id\":\"$math_id\"}")"
session_id="$(jq -er '.id' <<<"$session")"
curl -fsS -X POST "$api_base/study-sessions/$session_id/finish" "${auth[@]}" -d '{"duration_minutes":45,"focus_score":4,"mastery_score":3,"notes":"CI验收"}' >/dev/null
tasks="$(curl -fsS "$api_base/tasks?date=$today" "${auth[@]}")"
jq -e --arg id "$task_id" 'any(.[]; .id == $id and .actual_minutes == 45)' <<<"$tasks" >/dev/null

ielts="$(curl -fsS -X POST "$api_base/goals" "${auth[@]}" -d "{\"name\":\"雅思验收\",\"type\":\"ielts\",\"start_date\":\"$today\",\"exam_date\":\"$exam_date\",\"target_score\":7,\"settings\":{\"current_score\":5.5,\"listening_target\":7,\"reading_target\":7.5,\"writing_target\":6.5,\"speaking_target\":6.5,\"daily_vocabulary\":80}}")"
ielts_id="$(jq -er '.id' <<<"$ielts")"
ielts_phases="$(curl -fsS "$api_base/goals/$ielts_id/phases" "${auth[@]}")"
jq -e 'length == 6' <<<"$ielts_phases" >/dev/null
assessment="$(curl -fsS -X POST "$api_base/assessments" "${auth[@]}" -d "{\"goal_id\":\"$ielts_id\",\"name\":\"雅思模考\",\"assessment_type\":\"ielts\",\"exam_date\":\"$today\",\"score\":7,\"full_score\":9,\"duration_minutes\":170,\"details\":{\"listening\":7,\"reading\":7.5,\"writing\":6.5,\"speaking\":6.5}}")"
jq -e '.details.listening == 7 and .details.reading == 7.5' <<<"$assessment" >/dev/null

review="$(curl -fsS -X POST "$api_base/weekly-reviews" "${auth[@]}" -d "{\"goal_id\":\"$goal_id\",\"week_start\":\"$week_start\",\"planned_minutes\":300,\"actual_minutes\":240,\"completion_rate\":80,\"summary\":\"执行稳定\",\"problems\":\"任务偏大\",\"next_week_focus\":\"概率论\"}")"
jq -e --arg week "$week_start" '.week_start == $week' <<<"$review" >/dev/null

dashboard="$(curl -fsS "$api_base/dashboard" "${auth[@]}")"
analytics="$(curl -fsS "$api_base/analytics/overview" "${auth[@]}")"
jq -e '.today_minutes == 45 and .week_minutes >= 45' <<<"$dashboard" >/dev/null
jq -e '.total_minutes == 45 and .average_session_minutes == 45' <<<"$analytics" >/dev/null

echo "StudyTrack MVP smoke test passed"
