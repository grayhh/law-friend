package prompt

import (
	"fmt"
	"strings"

	"github.com/chan/lawmate/internal/precedents"
)

const SystemPrompt = `너는 "로메이트"이야. 법률 무지 때문에 어려움을 겪는 일반인들에게 친구처럼 편하게 법률 정보를 알려주는 동반자야.

원칙:
1. 친근하고 다정한 말투를 써. 딱딱한 법률 용어는 일상 언어로 풀어서 설명해줘.
2. 답변은 반드시 [참고 판례] 섹션에 제공된 판례에만 근거해야 해. 제공되지 않은 판례, 사건번호, 법령 조항을 절대 만들어내지 마.
3. 판례를 인용할 때는 사건번호(예: 2020다12345)와 사건명을 함께 표기해.
4. 사용자 상황에 대한 정보가 부족하면, 단정 짓지 말고 "혹시 ~한 상황이세요?" 같이 자연스러운 후속 질문을 던져.
5. 답변 끝에는 반드시 다음 면책 문구를 줄 바꿈 후 그대로 붙여:
   ---
   ⚖️ 로메이트의 답변은 일반적인 법률 정보일 뿐, 변호사의 법률 자문이 아니에요. 실제 사건에서는 반드시 전문 변호사와 상담하세요.

[참고 판례]가 비어 있거나 사용자 상황과 동떨어져 보이면, 솔직하게 "관련된 판례를 못 찾았어"라고 말하고 일반적인 방향 정도만 안내해.`

type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

func Build(history []Message, userQuery string, ctxPrecedents []precedents.Precedent) string {
	var sb strings.Builder

	sb.WriteString(SystemPrompt)
	sb.WriteString("\n\n")

	sb.WriteString("[참고 판례]\n")
	if len(ctxPrecedents) == 0 {
		sb.WriteString("(검색된 판례 없음)\n")
	} else {
		for i, p := range ctxPrecedents {
			fmt.Fprintf(&sb, "\n--- 판례 %d ---\n", i+1)
			fmt.Fprintf(&sb, "사건번호: %s\n", p.CaseNumber)
			fmt.Fprintf(&sb, "사건명: %s\n", p.CaseName)
			if p.CaseKind != "" {
				fmt.Fprintf(&sb, "사건종류: %s\n", p.CaseKind)
			}
			fmt.Fprintf(&sb, "법원: %s (%s 선고)\n", p.Court, p.DecisionDate)
			if p.Issues != "" {
				fmt.Fprintf(&sb, "판시사항: %s\n", truncate(p.Issues, 800))
			}
			if p.Summary != "" {
				fmt.Fprintf(&sb, "판결요지: %s\n", truncate(p.Summary, 1200))
			}
			if p.ReferencesLaw != "" {
				fmt.Fprintf(&sb, "참조조문: %s\n", truncate(p.ReferencesLaw, 400))
			}
		}
	}
	sb.WriteString("\n")

	if len(history) > 0 {
		sb.WriteString("[지금까지의 대화]\n")
		for _, m := range history {
			role := "사용자"
			if m.Role == "assistant" {
				role = "로메이트"
			}
			fmt.Fprintf(&sb, "%s: %s\n", role, m.Content)
		}
		sb.WriteString("\n")
	}

	sb.WriteString("[이번 질문]\n")
	sb.WriteString("사용자: ")
	sb.WriteString(userQuery)
	sb.WriteString("\n\n로메이트:")

	return sb.String()
}

func truncate(s string, max int) string {
	rs := []rune(s)
	if len(rs) <= max {
		return s
	}
	return string(rs[:max]) + "…"
}
