package main

import (
	"embed"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"time"

	"github.com/chan/lawmate/internal/precedents"
	"github.com/chan/lawmate/internal/server"
	"github.com/chan/lawmate/internal/sessions"
)

//go:embed web/*
var webEmbed embed.FS

//go:embed internal/data/dummy_precedents.json
var precedentsJSON []byte

func main() {
	if len(os.Args) < 2 {
		usage()
		os.Exit(2)
	}

	switch os.Args[1] {
	case "serve":
		serveCmd(os.Args[2:])
	case "-h", "--help", "help":
		usage()
	default:
		fmt.Fprintf(os.Stderr, "unknown command: %s\n\n", os.Args[1])
		usage()
		os.Exit(2)
	}
}

func usage() {
	fmt.Fprintln(os.Stderr, `로메이트 (lawmate) — 친구처럼 편한 법률 친구

Usage:
  lawmate serve [flags]   웹 UI를 띄워서 로메이트와 대화

Flags for 'serve':
  -addr string    서버 주소 (default ":8787")
  -data string    판례 JSON 파일 경로 (생략 시 디스크 자동 탐색 후 내장 fallback)
  -no-open        브라우저 자동 오픈 끄기
`)
}

func serveCmd(args []string) {
	fs := flag.NewFlagSet("serve", flag.ExitOnError)
	addr := fs.String("addr", ":8787", "서버 주소")
	dataPath := fs.String("data", "", "판례 JSON 파일 경로 (비우면 내장 데이터)")
	noOpen := fs.Bool("no-open", false, "브라우저 자동 오픈 끄기")
	_ = fs.Parse(args)

	store, err := loadStore(*dataPath)
	if err != nil {
		log.Fatalf("판례 로드 실패: %v", err)
	}

	sessPath, err := sessions.DefaultPath()
	if err != nil {
		log.Fatalf("세션 저장 경로 결정 실패: %v", err)
	}
	sessStore, err := sessions.OpenStore(sessPath)
	if err != nil {
		log.Fatalf("세션 저장소 열기 실패: %v", err)
	}
	log.Printf("세션 파일: %s", sessPath)

	webFS := server.MustSub(webEmbed, "web")
	srv := server.New(store, sessStore, webFS)

	httpSrv := &http.Server{
		Addr:              *addr,
		Handler:           srv.Routes(),
		ReadHeaderTimeout: 10 * time.Second,
	}

	url := fmt.Sprintf("http://localhost%s", *addr)
	log.Printf("로메이트가 준비됐어요! → %s", url)
	if !*noOpen {
		go openBrowser(url)
	}
	if err := httpSrv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("server error: %v", err)
	}
}

func loadStore(explicitPath string) (precedents.Store, error) {
	if explicitPath != "" {
		log.Printf("판례 데이터: %s (--data 지정)", explicitPath)
		return precedents.LoadDummyStore(explicitPath)
	}
	// 디스크 자동 탐색: 현재 작업 디렉터리, 실행파일 옆 순으로 확인.
	for _, p := range candidateDataPaths() {
		if info, err := os.Stat(p); err == nil && !info.IsDir() {
			log.Printf("판례 데이터: %s (디스크에서 로드)", p)
			return precedents.LoadDummyStore(p)
		}
	}
	log.Printf("판례 데이터: 내장 데이터 사용 (fallback, %d bytes)", len(precedentsJSON))
	return precedents.LoadDummyStoreFromBytes(precedentsJSON)
}

func candidateDataPaths() []string {
	var paths []string
	if cwd, err := os.Getwd(); err == nil {
		paths = append(paths, filepath.Join(cwd, "data", "precedents.json"))
	}
	if exe, err := os.Executable(); err == nil {
		paths = append(paths, filepath.Join(filepath.Dir(exe), "data", "precedents.json"))
	}
	return paths
}

func openBrowser(url string) {
	time.Sleep(300 * time.Millisecond)
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", url)
	case "windows":
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", url)
	default:
		cmd = exec.Command("xdg-open", url)
	}
	_ = cmd.Start()
}
