package fs

import (
	_ "embed"
	"html/template"
)

var (
	//go:embed schema.json
	schema []byte

	// allFields defines the canonical field order used for output and display.
	allFields = []string{"file", "createdAt", "modifiedAt", "accessedAt", "changedAt", "size", "lines", "mode", "isLink", "isDir"}

	// heatableFields are fields that can meaningfully carry a heat map.
	// mode, file, isLink are excluded as they are categorical / not ordinal.
	heatableFields = map[string]struct{}{
		"size":       {},
		"lines":      {},
		"createdAt":  {},
		"modifiedAt": {},
		"accessedAt": {},
		"changedAt":  {},
	}

	// excludableDateFields are the fields that can be used with --exclude-date-field.
	excludableDateFields = map[string]struct{}{
		"createdAt":  {},
		"modifiedAt": {},
		"accessedAt": {},
		"changedAt":  {},
	}

	tableTemplate = template.Must(template.New("page").Parse(`<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="UTF-8" />
<meta name="viewport" content="width=device-width, initial-scale=1.0" />
<title>File Metadata</title>
<style>
  *, *::before, *::after { box-sizing: border-box; margin: 0; padding: 0; }

  body {
    background: #fff;
    color: #1a1a1a;
    font-family: monospace;
    font-size: 13px;
    padding: 32px;
  }

  table {
    width: 100%;
    border-collapse: collapse;
  }

  th {
    padding: 8px 12px;
    text-align: left;
    font-size: 11px;
    text-transform: uppercase;
    letter-spacing: 0.06em;
    color: #888;
    border-bottom: 2px solid #e0e0e0;
    white-space: nowrap;
    cursor: pointer;
    user-select: none;
  }

  th:hover { color: #333; }
  th.sort-asc, th.sort-desc { color: #1a1a1a; }

  th .sort-icon {
    margin-left: 4px;
    opacity: 0.3;
  }

  th:hover .sort-icon,
  th.sort-asc .sort-icon,
  th.sort-desc .sort-icon { opacity: 1; }

  tbody tr { border-bottom: 1px solid #f0f0f0; }
  tbody tr:hover { background: #fafafa; }
  tbody tr.is-dir { background: #f5f5f5; }
  tbody tr.is-dir:hover { background: #eeeeee; }
  tbody tr.is-dir td { color: #555; font-style: italic; }

  td {
    padding: 8px 12px;
    white-space: nowrap;
  }

  td.col-num  { text-align: right; color: #555; }
  td.col-date { color: #888; }
  td.col-mode { color: #aaa; }
  td.col-bool { text-align: center; color: #aaa; }
</style>
</head>
<body>

{{ if .Files }}
<table id="tbl">
  <thead>
    <tr>
      {{ range $i, $col := .Columns }}
      <th data-col="{{ $i }}" data-type="{{ $col.Type }}">{{ $col.Label }}<span class="sort-icon">⇅</span></th>
      {{ end }}
    </tr>
  </thead>
  <tbody>
    {{ range .Files }}
    <tr{{ if .IsDir }} class="is-dir"{{ end }}>
      {{ range .Cells }}
      <td class="{{ .Class }}" data-val="{{ .Val }}"{{ if .HeatColor }} style="background:{{ .HeatColor }}"{{ end }}>{{ .Display }}</td>
      {{ end }}
    </tr>
    {{ end }}
  </tbody>
</table>
{{ else }}
<p style="color:#888">No files found in input.</p>
{{ end }}

<script>
(function () {
  var tbl = document.getElementById('tbl');
  if (!tbl) return;

  var tbody = tbl.querySelector('tbody');
  var ths   = Array.from(tbl.querySelectorAll('th'));
  var sortCol = -1;
  var sortAsc = true;

  ths.forEach(function (th, ci) {
    th.addEventListener('click', function () {
      if (sortCol === ci) {
        sortAsc = !sortAsc;
      } else {
        sortCol = ci;
        sortAsc = true;
      }

      ths.forEach(function (h) {
        h.classList.remove('sort-asc', 'sort-desc');
        h.querySelector('.sort-icon').textContent = '⇅';
      });

      th.classList.add(sortAsc ? 'sort-asc' : 'sort-desc');
      th.querySelector('.sort-icon').textContent = sortAsc ? '↑' : '↓';

      var type = th.dataset.type;
      var rows = Array.from(tbody.querySelectorAll('tr'));

      rows.sort(function (a, b) {
        var av = a.querySelectorAll('td')[ci].dataset.val;
        var bv = b.querySelectorAll('td')[ci].dataset.val;
        var cmp = type === 'number' ? Number(av) - Number(bv) : av.localeCompare(bv);
        return sortAsc ? cmp : -cmp;
      });

      rows.forEach(function (r) { tbody.appendChild(r); });
    });
  });
})();
</script>

</body>
</html>
`))
)
