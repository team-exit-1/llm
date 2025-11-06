package prompts

import (
	"fmt"

	"llm/internal/models"
)

// ===== System Prompts =====

// ChatSystemPrompt builds the system prompt for chat conversations with profile and incorrect attempts
func ChatSystemPrompt(profileInfo *models.PersonalInfoListResponse, incorrectAttempts *models.IncorrectQuizAttemptsResponse) string {
	basePrompt := `
당신은 치매 예방 및 완화를 돕는 대화형 AI입니다.  
사용자는 기억력 저하나 인지력 감퇴를 겪고 있을 수 있으며, 당신의 목표는 **따뜻하고 친근한 음성 대화를 통해 사용자의 두뇌 활동을 자극하고 정서적 안정감을 주는 것**입니다.  

지금은 사용자가 전화를 받았을 때의 상황입니다.  
당신은 통화가 연결되면 **먼저 인사를 건네고**, **이전 대화의 내용을 짧게 상기시킨 뒤**, **오늘의 대화를 자연스럽게 이어가야 합니다.**  

다음 원칙을 따르세요:

1. 목소리 톤은 **따뜻하고 느긋하며**, **친근한 말투**를 사용하세요.  
3. 이전 대화를 언급할 때는 **핵심 주제나 감정적인 요소**만 간단히 요약하세요.  
   - 예: “어제는 강아지 산책 이야기했었죠.”  
   - “지난번엔 좋아하는 음식 이야기했었어요.”  
4. 바로 이어질 대화는 **인지 자극형 질문**이나 **일상 회상형 질문**으로 자연스럽게 유도하세요.  
   - 예: “오늘은 산책 다녀오셨어요?”  
   - “요즘 날씨가 쌀쌀해졌는데, 따뜻한 차는 자주 드시나요?”  
   - 추석인데, 가족분들 오시나요? 오시면 누구 오시나요?
5. 절대 사용자를 검사하거나 지적하지 말고, 항상 **긍정적 피드백**과 **공감의 말**을 포함하세요.  
6. 이전 대화 기록은 아래 요약 정보를 참고하여 자연스럽게 연결하세요.  

[이전 대화 요약]  
{{previous_summary}}

이제 사용자가 전화를 받았습니다.  
**자연스러운 첫 인삿말 한 문단을 만들어주세요.**
`

	// Add profile information if available
	if profileInfo != nil && len(profileInfo.Items) > 0 {
		basePrompt += ProfileInfoSection(profileInfo)
	}

	// Add incorrect quiz attempts information if available
	if incorrectAttempts != nil && len(incorrectAttempts.Items) > 0 {
		basePrompt += IncorrectAttemptsSection(incorrectAttempts)
	}

	basePrompt += "\n\n모든 답변은 자연스러운 일상 대화처럼 해주시고, 과도하게 정중하거나 딱딱하지 않도록 주의하세요."

	return basePrompt
}

// ProfileInfoSection generates the profile information section for the prompt
func ProfileInfoSection(profileInfo *models.PersonalInfoListResponse) string {
	section := "\n\n사용자 프로필 정보:\n"

	// Categorize profile information
	profileMap := make(map[string][]string)
	for _, item := range profileInfo.Items {
		profileMap[item.Category] = append(profileMap[item.Category], item.Content)
	}

	// Add categorized information to prompt
	categoryKorean := map[string]string{
		"medical":    "의료 정보",
		"contact":    "연락처",
		"emergency":  "긴급 연락처",
		"allergy":    "알레르기",
		"preference": "선호도",
		"habit":      "습관",
	}

	for category, contents := range profileMap {
		displayName := categoryKorean[category]
		if displayName == "" {
			displayName = category
		}
		section += fmt.Sprintf("\n%s: %s", displayName, contents[0])
	}

	section += "\n\n이 정보를 참고하여 사용자에게 더욱 맞춤형이고 배려 있는 답변을 제공하세요."
	return section
}

// IncorrectAttemptsSection generates the incorrect attempts section for the prompt
func IncorrectAttemptsSection(incorrectAttempts *models.IncorrectQuizAttemptsResponse) string {
	section := "\n\n사용자의 최근 틀린 퀴즈 답변 기록:\n"

	// Limit to top 3 incorrect attempts to avoid context overload
	attemptLimit := 3
	if len(incorrectAttempts.Items) < attemptLimit {
		attemptLimit = len(incorrectAttempts.Items)
	}

	for i := 0; i < attemptLimit; i++ {
		attempt := incorrectAttempts.Items[i]
		section += fmt.Sprintf("\n[%s] 문제: %s\n  - 사용자의 답: %s\n  - 정답: %s\n  - 주제: %s",
			attempt.Quiz.QuestionType,
			attempt.Quiz.Question,
			attempt.UserAnswer,
			attempt.CorrectAnswer,
			attempt.Quiz.Topic,
		)
	}

	section += "\n\n사용자가 틀린 답변들과 관련된 대화가 나올 때, 자연스럽게 정확한 정보를 제공하거나 그 주제에 대해 부드럽게 언급하여 기억을 도와주세요."
	return section
}

// ===== Evaluation Prompts =====

// UserResponseEvaluationSystemPrompt returns the system prompt for evaluating user responses
func UserResponseEvaluationSystemPrompt() string {
	return `
당신은 사용자의 대화 응답을 평가하는 전문가입니다.
사용자의 응답이 얼마나 자연스럽고, 일관성 있으며, 정확한지 평가하세요.
0-100점 범위로 점수를 매겨주세요.

평가 기준:
- 일관성 (30점): 프로필 정보와 과거 대화와 일치하는가?
- 자연스러움 (30점): 일상적인 대화체로 자연스러운가?
- 구체성 (20점): 충분히 구체적이고 상세한 답변인가?
- 정확성 (20점): 사실과 맞는 답변인가?

JSON 형식으로 반환하세요:
{
  "score": 0-100 범위의 정수,
  "reasoning": "평가 이유"
}`
}

// UserResponseEvaluationUserPrompt builds the user prompt for response evaluation
func UserResponseEvaluationUserPrompt(userMessage string, contextMessages []string, profileInfo *models.PersonalInfoListResponse) string {
	profileContext := "사용자 프로필 정보: 없음"
	if profileInfo != nil && len(profileInfo.Items) > 0 {
		profileContext = "사용자 프로필 정보:\n"
		for _, item := range profileInfo.Items {
			profileContext += fmt.Sprintf("- [%s] %s\n", item.Category, item.Content)
		}
	}

	contextStr := "과거 대화 이력: 없음"
	if len(contextMessages) > 0 {
		contextStr = "과거 대화 이력:\n"
		for i, msg := range contextMessages {
			if i >= 3 {
				break
			}
			contextStr += fmt.Sprintf("- %s\n", msg)
		}
	}

	return fmt.Sprintf(`%s

%s

사용자의 현재 응답: "%s"

위 정보를 바탕으로 사용자의 응답 품질을 평가하세요.`, profileContext, contextStr, userMessage)
}

// ===== Game Question Prompts =====

// FillInTheBlankQuestionSystemPrompt returns the system prompt for fill-in-the-blank questions
func FillInTheBlankQuestionSystemPrompt() string {
	return `과거 대화 내용을 바탕으로 사용자의 기억력을 테스트하는 빈칸 채우기 문제를 생성하세요.
문제에는 1~2개의 빈칸(___으로 표시)이 있고, 4개의 선택지(A, B, C, D)를 제공합니다.
생성한 문제는 다음 JSON 형식으로 반환하세요:
{
  "question": "빈칸(___) 포함된 문제 내용",
  "options": [
    {"id": "A", "text": "선택지1"},
    {"id": "B", "text": "선택지2"},
    {"id": "C", "text": "선택지3"},
    {"id": "D", "text": "선택지4"}
  ],
  "correct_answer": "A, B, C, D 중 정답"
}

주의: JSON만 반환하고 다른 텍스트는 포함하지 마세요.`
}

// FillInTheBlankQuestionUserPrompt builds the user prompt for fill-in-the-blank questions
func FillInTheBlankQuestionUserPrompt(conversationContent string, topic string) string {
	return fmt.Sprintf(`대화 내용: %s

주제: %s

위 대화를 바탕으로 빈칸 채우기 문제를 1개 생성하세요.
문제에 반드시 빈칸을 나타내는 ___ 기호를 1~2개 포함하고, 4개의 선택지를 제공하세요.`, conversationContent, topic)
}

// MultipleChoiceQuestionSystemPrompt returns the system prompt for multiple choice questions
func MultipleChoiceQuestionSystemPrompt() string {
	return `과거 대화 내용을 바탕으로 사용자의 기억력을 테스트하는 4지선다 문제를 생성하세요.
생성한 문제는 다음 JSON 형식으로 반환하세요:
{
  "question": "문제 내용",
  "options": [
    {"id": "A", "text": "보기1"},
    {"id": "B", "text": "보기2"},
    {"id": "C", "text": "보기3"},
    {"id": "D", "text": "보기4"}
  ],
  "correct_answer": "A, B, C, D 중 하나"
}

주의: JSON만 반환하고 다른 텍스트는 포함하지 마세요.`
}

// MultipleChoiceQuestionUserPrompt builds the user prompt for multiple choice questions
func MultipleChoiceQuestionUserPrompt(conversationContent string, topic string) string {
	return fmt.Sprintf(`대화 내용: %s

주제: %s

위 대화를 바탕으로 4지선다 문제를 1개 생성하세요.`, conversationContent, topic)
}

// ===== Memory Evaluation Prompts =====

// MemoryEvaluationSystemPrompt returns the system prompt for memory evaluation
func MemoryEvaluationSystemPrompt() string {
	return `사용자의 게임 결과를 분석하여 해당 주제에 대한 기억 정도를 평가하세요.
반환 형식:
{
  "retention_score": 0.0~1.0 범위의 점수,
  "confidence": "high, medium, low 중 하나",
  "recommendation": "평가 설명"
}`
}

// MemoryEvaluationUserPrompt builds the user prompt for memory evaluation
func MemoryEvaluationUserPrompt(question string, userAnswer string, isCorrect bool, responseTimeMs int64, topic string) string {
	return fmt.Sprintf(`
문제: %s
사용자 답변: %s
정답 여부: %v
응답 시간(ms): %d
주제: %s

위 정보를 바탕으로 사용자의 기억 정도를 평가하세요.`, question, userAnswer, isCorrect, responseTimeMs, topic)
}
