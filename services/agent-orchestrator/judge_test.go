package orchestrator

import (
	"context"
	"errors"
	"strings"
	"testing"
)

func TestParseJudgeResponseJSON(t *testing.T) {
	jr, err := ParseJudgeResponse(`{"pass":true,"score":0.9,"rationale":"good"}`)
	if err != nil || !jr.Pass || jr.Score != 0.9 {
		t.Fatalf("%+v %v", jr, err)
	}
	jr, err = ParseJudgeResponse("Verdict: FAIL\nSCORE: 0.1")
	if err != nil || jr.Pass || jr.Score != 0.1 {
		t.Fatalf("%+v %v", jr, err)
	}
}

func TestGatewayJudgeWithStub(t *testing.T) {
	gw := &stubGateway{content: `{"pass":true,"score":1.0,"rationale":"meets rubric"}`}
	j := &GatewayJudge{Client: gw, Model: "m"}
	pass, score, rat, err := j.Judge(context.Background(), "must mention atlas", "atlas is great")
	if err != nil || !pass || score != 1.0 || !strings.Contains(rat, "meets") {
		t.Fatalf("%v %v %q %v", pass, score, rat, err)
	}
}

func TestStaticJudgeError(t *testing.T) {
	j := StaticJudge{Err: errors.New("down")}
	_, _, _, err := j.Judge(context.Background(), "r", "c")
	if err == nil {
		t.Fatal("expected error")
	}
}
