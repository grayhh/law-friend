const chatEl = document.getElementById('chat');
const formEl = document.getElementById('composer');
const inputEl = document.getElementById('input');
const sendBtn = document.getElementById('sendBtn');
const cliPicker = document.getElementById('cliPicker');
const sessionListEl = document.getElementById('sessionList');
const newChatBtn = document.getElementById('newChatBtn');
const chatTitle = document.getElementById('chatTitle');
const readonlyBanner = document.getElementById('readonlyBanner');
const startNewFromArchive = document.getElementById('startNewFromArchive');
const modal = document.getElementById('precedentModal');
const modalBody = document.getElementById('modalBody');
const modalClose = document.getElementById('modalClose');

const state = {
  history: [],
  selectedCLI: null,
  availableCLI: [],
  busy: false,
  sessionID: null,     // null = unsent new session, otherwise active session id
  readonly: false,     // true when viewing an archived session
  activeSession: null, // currently selected sidebar session id
};

const INTRO_HTML = `안녕! 나는 <strong>법친</strong>이야 👋<br />
법 때문에 헷갈리거나 답답한 일이 있으면 편하게 물어봐.<br />
어떤 상황인지 그냥 말로 풀어줘도 돼.`;

async function init() {
  try {
    const res = await fetch('/api/cli/available');
    const data = await res.json();
    state.availableCLI = data.available || [];
  } catch (_) {
    state.availableCLI = [];
  }

  for (const btn of cliPicker.querySelectorAll('.cli-btn')) {
    const cli = btn.dataset.cli;
    if (!state.availableCLI.includes(cli)) {
      btn.disabled = true;
      btn.title = `${cli} CLI가 설치되어 있지 않아요`;
    }
    btn.addEventListener('click', () => {
      if (state.readonly) return;
      selectCLI(cli);
    });
  }

  if (state.availableCLI.length > 0) {
    selectCLI(state.availableCLI[0]);
  } else {
    showIntro();
    appendAssistant({
      answer: 'Claude CLI 또는 Codex CLI가 설치되어 있지 않아 답변을 만들 수 없어요. 둘 중 하나를 먼저 설치하고 인증해줘.',
      precedents: [],
    });
  }

  await refreshSessionList();
  setLiveMode();
}

function selectCLI(cli) {
  state.selectedCLI = cli;
  for (const btn of cliPicker.querySelectorAll('.cli-btn')) {
    btn.classList.toggle('active', btn.dataset.cli === cli);
  }
}

/* ---------- Mode handling ---------- */
function setLiveMode() {
  state.readonly = false;
  state.sessionID = null;
  state.history = [];
  state.activeSession = null;
  chatTitle.textContent = '새 대화';
  readonlyBanner.classList.add('hidden');
  formEl.classList.remove('hidden');
  inputEl.disabled = false;
  sendBtn.disabled = false;
  for (const btn of cliPicker.querySelectorAll('.cli-btn')) {
    btn.disabled = !state.availableCLI.includes(btn.dataset.cli);
  }
  chatEl.innerHTML = '';
  showIntro();
  highlightActiveSession();
}

function setReadonlyMode(session) {
  state.readonly = true;
  state.sessionID = session.id;
  state.activeSession = session.id;
  state.history = session.messages.map(m => ({ role: m.role, content: m.content }));
  chatTitle.textContent = session.title;
  readonlyBanner.classList.remove('hidden');
  formEl.classList.add('hidden');
  for (const btn of cliPicker.querySelectorAll('.cli-btn')) {
    btn.disabled = true;
    btn.classList.toggle('active', btn.dataset.cli === session.cli);
  }
  chatEl.innerHTML = '';
  for (const msg of session.messages) {
    if (msg.role === 'user') renderUser(msg.content);
    else renderAssistant(msg.content, msg.precedents || []);
  }
  highlightActiveSession();
}

function showIntro() {
  const el = document.createElement('div');
  el.className = 'message assistant intro';
  const bubble = document.createElement('div');
  bubble.className = 'bubble';
  bubble.innerHTML = INTRO_HTML;
  el.appendChild(bubble);
  chatEl.appendChild(el);
}

/* ---------- Chat send ---------- */
formEl.addEventListener('submit', async (e) => {
  e.preventDefault();
  if (state.busy || state.readonly) return;
  const text = inputEl.value.trim();
  if (!text) return;
  if (!state.selectedCLI) {
    alert('사용할 CLI를 먼저 선택해줘');
    return;
  }

  const isFirst = state.history.length === 0;
  appendUser(text);
  inputEl.value = '';
  setBusy(true);

  const typing = appendTyping();

  try {
    const res = await fetch('/api/chat', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({
        query: text,
        history: state.history.slice(0, -1),
        cli: state.selectedCLI,
        session_id: state.sessionID || '',
      }),
    });
    typing.remove();
    if (!res.ok) {
      const errText = await res.text();
      appendAssistant({ answer: `❌ 오류: ${errText}`, precedents: [] });
      return;
    }
    const data = await res.json();
    state.sessionID = data.session_id;
    appendAssistant(data);
    if (isFirst) {
      chatTitle.textContent = text.length > 30 ? text.slice(0, 30) + '…' : text;
      await refreshSessionList();
    } else {
      await refreshSessionList();
    }
  } catch (err) {
    typing.remove();
    appendAssistant({ answer: `❌ 네트워크 오류: ${err.message}`, precedents: [] });
  } finally {
    setBusy(false);
  }
});

function appendUser(text) {
  state.history.push({ role: 'user', content: text });
  renderUser(text);
}

function renderUser(text) {
  const el = document.createElement('div');
  el.className = 'message user';
  const bubble = document.createElement('div');
  bubble.className = 'bubble';
  bubble.textContent = text;
  el.appendChild(bubble);
  chatEl.appendChild(el);
  scrollToBottom();
}

function appendAssistant({ answer, precedents }) {
  state.history.push({ role: 'assistant', content: answer });
  renderAssistant(answer, precedents);
}

function renderAssistant(text, precedents) {
  const el = document.createElement('div');
  el.className = 'message assistant';
  const bubble = document.createElement('div');
  bubble.className = 'bubble';
  const md = document.createElement('div');
  md.className = 'markdown';
  md.innerHTML = DOMPurify.sanitize(marked.parse(text || ''));
  bubble.appendChild(md);

  if (precedents && precedents.length > 0) {
    const cited = document.createElement('div');
    cited.className = 'cited';
    const label = document.createElement('span');
    label.textContent = '📚 참고 판례:';
    cited.appendChild(label);
    for (const p of precedents) {
      const link = document.createElement('button');
      link.className = 'cited-link';
      link.textContent = `${p.case_number} ${p.case_name}`;
      link.addEventListener('click', () => openPrecedent(p));
      cited.appendChild(link);
    }
    bubble.appendChild(cited);
  }

  el.appendChild(bubble);
  chatEl.appendChild(el);
  scrollToBottom();
}

function appendTyping() {
  const el = document.createElement('div');
  el.className = 'message assistant';
  const bubble = document.createElement('div');
  bubble.className = 'bubble typing';
  bubble.textContent = '법친이 판례를 찾아보는 중…';
  el.appendChild(bubble);
  chatEl.appendChild(el);
  scrollToBottom();
  return el;
}

function setBusy(b) {
  state.busy = b;
  sendBtn.disabled = b;
  inputEl.disabled = b;
}

function scrollToBottom() {
  chatEl.scrollTo({ top: chatEl.scrollHeight, behavior: 'smooth' });
}

/* ---------- Precedent modal ---------- */
async function openPrecedent(p) {
  // Try server first; fall back to embedded precedent object (for archived sessions).
  let detail = p;
  try {
    const res = await fetch(`/api/precedents/${encodeURIComponent(p.id)}`);
    if (res.ok) detail = await res.json();
  } catch (_) { /* use fallback */ }

  modalBody.innerHTML = '';
  const title = document.createElement('h3');
  title.textContent = `${detail.case_number} ${detail.case_name}`;
  const meta = document.createElement('div');
  meta.className = 'meta';
  meta.textContent = `${detail.court} · ${detail.decision_date}`;
  const pre = document.createElement('pre');
  pre.textContent = detail.full_text || detail.summary || '(원문 없음)';
  modalBody.appendChild(title);
  modalBody.appendChild(meta);
  modalBody.appendChild(pre);
  modal.classList.remove('hidden');
}

modalClose.addEventListener('click', () => modal.classList.add('hidden'));
modal.addEventListener('click', (e) => {
  if (e.target === modal) modal.classList.add('hidden');
});

inputEl.addEventListener('keydown', (e) => {
  if (e.key === 'Enter' && !e.shiftKey) {
    e.preventDefault();
    formEl.requestSubmit();
  }
});

/* ---------- Sessions sidebar ---------- */
newChatBtn.addEventListener('click', () => setLiveMode());
startNewFromArchive.addEventListener('click', () => setLiveMode());

async function refreshSessionList() {
  try {
    const res = await fetch('/api/sessions');
    const data = await res.json();
    renderSessionList(data.sessions || []);
  } catch (err) {
    sessionListEl.innerHTML = `<div class="session-empty">목록 불러오기 실패</div>`;
  }
}

function renderSessionList(sessions) {
  sessionListEl.innerHTML = '';
  if (sessions.length === 0) {
    const empty = document.createElement('div');
    empty.className = 'session-empty';
    empty.textContent = '아직 대화가 없어요';
    sessionListEl.appendChild(empty);
    return;
  }
  for (const s of sessions) {
    const item = document.createElement('div');
    item.className = 'session-item';
    item.dataset.id = s.id;
    if (s.id === state.activeSession) item.classList.add('active');

    const title = document.createElement('div');
    title.className = 'session-title';
    title.textContent = s.title;

    const meta = document.createElement('div');
    meta.className = 'session-meta';
    const badge = document.createElement('span');
    badge.className = 'cli-badge';
    badge.textContent = s.cli;
    const time = document.createElement('span');
    time.textContent = formatTime(s.updated_at);
    meta.appendChild(badge);
    meta.appendChild(time);

    const del = document.createElement('button');
    del.className = 'session-delete';
    del.textContent = '×';
    del.title = '대화 삭제';
    del.addEventListener('click', async (e) => {
      e.stopPropagation();
      if (!confirm('이 대화를 삭제할까요?')) return;
      await fetch(`/api/sessions/${encodeURIComponent(s.id)}`, { method: 'DELETE' });
      if (state.activeSession === s.id) setLiveMode();
      await refreshSessionList();
    });

    item.appendChild(title);
    item.appendChild(meta);
    item.appendChild(del);

    item.addEventListener('click', () => loadSession(s.id));
    sessionListEl.appendChild(item);
  }
}

function highlightActiveSession() {
  for (const item of sessionListEl.querySelectorAll('.session-item')) {
    item.classList.toggle('active', item.dataset.id === state.activeSession);
  }
}

async function loadSession(id) {
  try {
    const res = await fetch(`/api/sessions/${encodeURIComponent(id)}`);
    if (!res.ok) {
      alert('대화를 불러오지 못했어요');
      return;
    }
    const sess = await res.json();
    setReadonlyMode(sess);
  } catch (err) {
    alert('오류: ' + err.message);
  }
}

function formatTime(iso) {
  try {
    const d = new Date(iso);
    const now = new Date();
    const sameDay = d.toDateString() === now.toDateString();
    if (sameDay) {
      return d.toLocaleTimeString('ko-KR', { hour: '2-digit', minute: '2-digit' });
    }
    return d.toLocaleDateString('ko-KR', { month: '2-digit', day: '2-digit' });
  } catch (_) { return ''; }
}

init();
