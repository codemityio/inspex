<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="UTF-8" />
  <meta name="viewport" content="width=device-width, initial-scale=1.0" />
  <title>git log viewer</title>
  <script src="https://unpkg.com/vue@3/dist/vue.global.prod.js"></script>
  <style>
    @import url('https://fonts.googleapis.com/css2?family=JetBrains+Mono:wght@400;600&display=swap');
    *, *::before, *::after { box-sizing: border-box; margin: 0; padding: 0; }
    body { background: #f6f8fa; color: #1f2328; font-family: 'JetBrains Mono', monospace; min-height: 100vh; font-size: 13px; }
    ::-webkit-scrollbar { width: 6px; height: 6px; }
    ::-webkit-scrollbar-track { background: #f6f8fa; }
    ::-webkit-scrollbar-thumb { background: #d0d7de; border-radius: 3px; }

    .toolbar {
      padding: 14px 32px;
      border-bottom: 1px solid #d0d7de;
      background: #ffffff;
      display: flex;
      align-items: center;
      gap: 16px;
      position: sticky;
      top: 0;
      z-index: 10;
      box-shadow: 0 1px 0 #d0d7de;
    }
    .search-wrap { position: relative; max-width: 400px; flex: 1; }
    .search-icon { position: absolute; left: 11px; top: 50%; transform: translateY(-50%); color: #8c959f; font-size: 15px; pointer-events: none; }
    .search-input {
      width: 100%; background: #f6f8fa; border: 1px solid #d0d7de;
      border-radius: 6px; padding: 7px 12px 7px 32px;
      color: #1f2328; font-family: 'JetBrains Mono', monospace; font-size: 12px;
      outline: none; transition: border-color 0.15s, box-shadow 0.15s;
    }
    .search-input:focus { border-color: #0969da; box-shadow: 0 0 0 3px rgba(9,105,218,0.1); background: #fff; }
    .toolbar-meta { color: #8c959f; font-size: 11px; white-space: nowrap; }

    .table-wrap { padding: 20px 32px 80px; }
    table { width: 100%; border-collapse: collapse; table-layout: fixed; }
    col.col-file    { width: 26%; }
    col.col-dates   { width: 26%; }
    col.col-commits { width: 48%; }
    thead th {
      text-align: left; padding: 8px 10px; font-size: 10px; font-weight: 600;
      letter-spacing: 1px; text-transform: uppercase; color: #8c959f;
      border-bottom: 1px solid #d0d7de; background: #f6f8fa;
    }
    tbody tr { border-bottom: 1px solid #eaeef2; vertical-align: top; animation: fadeIn 0.2s ease both; }
    tbody tr:hover { background: #f6f8fa; }
    tbody td { padding: 12px 10px; }
    @keyframes fadeIn { from { opacity: 0; transform: translateY(4px); } to { opacity: 1; transform: translateY(0); } }

    .file-path-dir  { color: #8c959f; font-size: 12px; }
    .file-path-name { color: #0969da; font-size: 12px; font-weight: 600; }
    .file-count { margin-top: 3px; font-size: 10px; color: #8c959f; }

    .dates-cell { display: flex; flex-wrap: wrap; gap: 4px; }
    .date-tag {
      display: inline-flex; flex-direction: column; align-items: flex-start;
      background: #ffffff; border: 1px solid #d0d7de;
      border-radius: 6px; padding: 4px 9px; cursor: default;
      transition: border-color 0.15s, box-shadow 0.15s;
    }
    .date-tag:hover { border-color: #0969da; box-shadow: 0 0 0 3px rgba(9,105,218,0.08); }
    .date-tag-date { color: #1f2328; font-size: 11px; font-weight: 600; }
    .date-tag-time { color: #8c959f; font-size: 10px; margin-top: 1px; }

    .commit-row {
      display: flex; align-items: center; gap: 8px;
      padding: 4px 7px; border-radius: 6px; cursor: pointer;
      transition: background 0.12s;
    }
    .commit-row:hover  { background: #eaeef2; }
    .commit-row.pinned { background: #ddf4ff; outline: 1px solid #54aeff; }
    .commit-hash {
      font-size: 11px; color: #0969da; background: #ddf4ff;
      border: 1px solid #54aeff; border-radius: 4px; padding: 1px 5px; flex-shrink: 0;
    }
    .commit-msg { color: #1f2328; font-size: 12px; flex: 1; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
    .stat-add { font-size: 10px; color: #1a7f37; flex-shrink: 0; font-weight: 600; }
    .stat-rem { font-size: 10px; color: #cf222e; flex-shrink: 0; font-weight: 600; }

    .diff-popup {
      position: fixed; z-index: 300; width: 560px;
      background: #ffffff; border: 1px solid #d0d7de; border-radius: 12px;
      box-shadow: 0 8px 24px rgba(140,149,159,0.2), 0 1px 3px rgba(0,0,0,0.12);
      font-family: 'JetBrains Mono', monospace; font-size: 12px;
      overflow: hidden;
      animation: popIn 0.15s cubic-bezier(.22,1,.36,1);
    }
    .diff-popup.floating { pointer-events: none; }
    .diff-popup.pinned   { pointer-events: all; cursor: default; box-shadow: 0 16px 48px rgba(140,149,159,0.3), 0 2px 6px rgba(0,0,0,0.15); }
    @keyframes popIn {
      from { opacity: 0; transform: translateY(-4px) scale(0.98); }
      to   { opacity: 1; transform: translateY(0) scale(1); }
    }
    .popup-header {
      padding: 10px 14px; background: #f6f8fa;
      border-bottom: 1px solid #d0d7de;
      display: flex; gap: 8px; align-items: center; flex-wrap: wrap;
    }
    .popup-pin-hint { margin-left: auto; font-size: 10px; color: #8c959f; flex-shrink: 0; }
    .popup-hash { font-size: 11px; background: #ddf4ff; color: #0969da; border: 1px solid #54aeff; border-radius: 4px; padding: 2px 7px; letter-spacing: 0.5px; flex-shrink: 0; }
    .popup-datetime { display: flex; flex-direction: column; gap: 1px; flex-shrink: 0; }
    .popup-date { color: #1f2328; font-size: 11px; font-weight: 600; }
    .popup-time { color: #8c959f; font-size: 10px; }
    .popup-author { font-size: 10px; border-radius: 20px; padding: 1px 7px; flex-shrink: 0; font-weight: 600; }
    .popup-msg { color: #1f2328; font-size: 11px; flex: 1; min-width: 0; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
    .popup-stats { padding: 5px 14px; border-bottom: 1px solid #eaeef2; display: flex; gap: 12px; font-size: 11px; background: #fff; }
    .popup-added { color: #1a7f37; font-weight: 600; }
    .popup-removed { color: #cf222e; font-weight: 600; }

    .diff-body  { max-height: 380px; overflow: auto; background: #fff; }
    .diff-lines { display: inline-block; min-width: 100%; vertical-align: top; }
    .hunk-header {
      display: block; padding: 4px 14px; white-space: pre; min-width: 100%;
      background: #f6f8fa; color: #0969da; font-size: 11px;
      border-top: 1px solid #eaeef2; border-bottom: 1px solid #eaeef2;
    }
    .hunk-context { color: #8c959f; margin-left: 8px; }
    .diff-line { display: flex; align-items: baseline; min-width: 100%; }
    .diff-line.added   { background: #e6ffec; }
    .diff-line.removed { background: #ffebe9; }
    .diff-prefix { width: 24px; min-width: 24px; text-align: center; flex-shrink: 0; user-select: none; line-height: 1.6; }
    .diff-line.added   .diff-prefix { color: #1a7f37; }
    .diff-line.removed .diff-prefix { color: #cf222e; }
    .diff-line.context .diff-prefix { color: #8c959f; }
    .diff-content { white-space: pre; padding-right: 14px; line-height: 1.6; }
    .diff-line.added   .diff-content { color: #1a7f37; }
    .diff-line.removed .diff-content { color: #cf222e; }
    .diff-line.context .diff-content { color: #1f2328; }

    .popup-overlay { position: fixed; inset: 0; z-index: 299; background: rgba(0,0,0,0.08); }
  </style>
</head>
<body>
  <div id="app"></div>
  <script>
    const data = [[.]];
  </script>
  <script>
    const { createApp, ref, computed, onMounted, onUnmounted, nextTick } = Vue;

    const colours = [
      ["#ddf4ff","#0969da"],["#dcffe4","#1a7f37"],["#fff8c5","#9a6700"],
      ["#ffebe9","#cf222e"],["#fbefff","#8250df"],["#fff1e5","#bc4c00"],
    ];

    const authorMap = {};

    let colorIdx = 0;

    function authorColor(name) {
      if (!authorMap[name]) authorMap[name] = colours[colorIdx++ % colours.length];
      return authorMap[name];
    }

    function uniqueDateTimes(changes) {
      return changes.map(c => ({
        date: c.date || (c.time ? c.time.slice(0, 10) : ""),
        time: c.time ? c.time.slice(11, 16) : "",
      }));
    }

    function timeLabel(iso) { return iso ? iso.slice(11, 16) : ""; }

    function fileParts(path) {
      const idx = path.lastIndexOf("/");
      if (idx === -1) return { dir: "", name: path };
      return { dir: path.slice(0, idx + 1), name: path.slice(idx + 1) };
    }

    const App = {
      setup() {
        const search  = ref("");
        const popup   = ref(null);
        const popupEl = ref(null);

        const files = computed(() =>
          Object.keys(data)
            .filter(f => !search.value || f.toLowerCase().includes(search.value.toLowerCase()))
            .sort()
        );
        const totalCommits = computed(() =>
          files.value.reduce((s, f) => s + data[f].length, 0)
        );

        function positionPopup(evt) {
          if (!popupEl.value) return;
          const pw = popupEl.value.offsetWidth  || 560;
          const ph = popupEl.value.offsetHeight || 300;
          const margin = 16, gap = 14;
          let x = evt.clientX + gap;
          let y = evt.clientY + gap;
          if (x + pw > window.innerWidth  - margin) x = evt.clientX - pw - gap;
          if (y + ph > window.innerHeight - margin) y = evt.clientY - ph - gap;
          if (x < margin) x = margin;
          if (y < margin) y = margin;
          popup.value.x = x;
          popup.value.y = y;
        }

        function onCommitEnter(evt, change, file) {
          if (popup.value && popup.value.pinned) return;
          popup.value = { change, file, x: 0, y: 0, pinned: false };
          nextTick(() => positionPopup(evt));
        }
        function onCommitLeave() {
          if (popup.value && !popup.value.pinned) popup.value = null;
        }
        function onMouseMove(evt) {
          if (popup.value && !popup.value.pinned) positionPopup(evt);
        }
        function onCommitClick(evt, change, file) {
          evt.stopPropagation();
          if (popup.value && popup.value.pinned && popup.value.change.hash === change.hash && popup.value.file === file) {
            popup.value = null;
            return;
          }
          popup.value = { change, file, x: 0, y: 0, pinned: true };
          nextTick(() => positionPopup(evt));
        }
        function onOverlayClick() {
          if (popup.value && popup.value.pinned) popup.value = null;
        }

        onMounted(()  => window.addEventListener("mousemove", onMouseMove));
        onUnmounted(() => window.removeEventListener("mousemove", onMouseMove));

        return {
          search, popup, popupEl, files, totalCommits, data,
          onCommitEnter, onCommitLeave, onCommitClick, onOverlayClick,
          authorColor, uniqueDateTimes, timeLabel, fileParts,
        };
      },

      template: `
        <div class="toolbar">
          <div class="search-wrap">
            <span class="search-icon">⌕</span>
            <input v-model="search" class="search-input" placeholder="Filter files…" />
          </div>
          <span class="toolbar-meta">
            {{ files.length }} file{{ files.length !== 1 ? 's' : '' }}
            &middot;
            {{ totalCommits }} commit{{ totalCommits !== 1 ? 's' : '' }}
          </span>
        </div>

        <div class="table-wrap">
          <table>
            <colgroup>
              <col class="col-file" /><col class="col-dates" /><col class="col-commits" />
            </colgroup>
            <thead>
              <tr>
                <th>File</th><th>Change Dates</th><th>Commits</th>
              </tr>
            </thead>
            <tbody>
              <tr v-for="(file, fi) in files" :key="file" :style="{ animationDelay: fi * 30 + 'ms' }">
                <td>
                  <div>
                    <span class="file-path-dir">{{ fileParts(file).dir }}</span><span class="file-path-name">{{ fileParts(file).name }}</span>
                  </div>
                  <div class="file-count">{{ data[file].length }} commit{{ data[file].length !== 1 ? 's' : '' }}</div>
                </td>
                <td>
                  <div class="dates-cell">
                    <span v-for="dt in uniqueDateTimes(data[file])" :key="dt.date" class="date-tag">
                      <span class="date-tag-date">{{ dt.date }}</span>
                      <span v-if="dt.time" class="date-tag-time">{{ dt.time }}</span>
                    </span>
                  </div>
                </td>
                <td>
                  <div
                    v-for="change in data[file]" :key="change.hash"
                    class="commit-row"
                    :class="{ pinned: popup && popup.pinned && popup.change.hash === change.hash && popup.file === file }"
                    @mouseenter="e => onCommitEnter(e, change, file)"
                    @mouseleave="onCommitLeave"
                    @click="e => onCommitClick(e, change, file)"
                  >
                    <span class="commit-hash">{{ change.hash }}</span>
                    <span class="commit-msg">{{ change.message }}</span>
                    <span class="stat-add" v-if="change.linesAdded != null">+{{ change.linesAdded }}</span>
                    <span class="stat-rem" v-if="change.linesRemoved != null">−{{ change.linesRemoved }}</span>
                  </div>
                </td>
              </tr>
            </tbody>
          </table>
        </div>

        <Teleport to="body">
          <div v-if="popup && popup.pinned" class="popup-overlay" @click="onOverlayClick"></div>
        </Teleport>

        <Teleport to="body">
          <div
            v-if="popup"
            ref="popupEl"
            class="diff-popup"
            :class="popup.pinned ? 'pinned' : 'floating'"
            :style="{ top: popup.y + 'px', left: popup.x + 'px' }"
          >
            <div class="popup-header">
              <span class="popup-hash">{{ popup.change.hash }}</span>
              <div class="popup-datetime">
                <span class="popup-date">{{ popup.change.date }}</span>
                <span class="popup-time" v-if="popup.change.time">{{ timeLabel(popup.change.time) }}</span>
              </div>
              <span class="popup-author" :style="{ background: authorColor(popup.change.author)[0], color: authorColor(popup.change.author)[1] }">
                {{ popup.change.author }}
              </span>
              <span class="popup-msg">{{ popup.change.message }}</span>
              <span class="popup-pin-hint">{{ popup.pinned ? 'click outside to close' : 'click to pin' }}</span>
            </div>
            <div class="popup-stats">
              <span class="popup-added"   v-if="popup.change.linesAdded   != null">+{{ popup.change.linesAdded }} added</span>
              <span class="popup-removed" v-if="popup.change.linesRemoved != null">−{{ popup.change.linesRemoved }} removed</span>
            </div>
            <div class="diff-body">
              <div class="diff-lines">
                <template v-for="(hunk, hi) in popup.change.diff" :key="hi">
                  <div class="hunk-header">
                    @@ -{{ hunk.fromLine }} +{{ hunk.toLine }} @@
                    <span class="hunk-context" v-if="hunk.context">{{ hunk.context }}</span>
                  </div>
                  <div v-for="(ch, ci) in hunk.changes" :key="ci" class="diff-line" :class="ch.type">
                    <span class="diff-prefix">{{ ch.type === 'added' ? '+' : ch.type === 'removed' ? '−' : ' ' }}</span>
                    <span class="diff-content">{{ ch.content }}</span>
                  </div>
                </template>
              </div>
            </div>
          </div>
        </Teleport>
      `
    };
    createApp(App).mount("#app");
  </script>
</body>
</html>
