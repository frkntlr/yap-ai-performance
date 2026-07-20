---
name: yap
description: >-
  Yap AI Performance orchestrator: health check, project context, graphify update,
  CodeGraphContext + Graphify MCP analysis, code-review and architecture answers.
  Use when the user types /yap, says yap, or asks about architecture, dependencies,
  callers/callees, complexity, dead code, performance, code review, project summary,
  MCP status, or wants automatic graph-backed answers without naming tools.
---

# /yap

Kullanıcı cümlesine göre Graphify + CodeGraphContext + `yap` CLI gücünü otomatik seç.

## 0) Intent router (cümleden anla)

| Kullanıcı niyeti (örnek) | Aksiyon |
|--------------------------|---------|
| sağlık, bozuldu, MCP, status | `yap status` |
| özet, bağlam, ne değişti, context | `yap context` veya `yap context --prompt --with-diff` |
| kod değişti / graph eski / update | `graphify update .` (gerekirse CGC `watch_directory` / `add_code_to_graph`) |
| mimari, nasıl çalışıyor, ilişki | Graphify `query_graph` / `god_nodes` / `graph_stats` |
| A→B yol, bağımlılık | Graphify `shortest_path` veya `graphify path` |
| kim çağırıyor, callee, hierarchy | CGC `analyze_code_relationships` |
| bul, nerede, snippet | CGC `find_code` |
| karmaşıklık, Big O hissi, hotspot | CGC `find_most_complex_functions` / `calculate_cyclomatic_complexity` |
| ölü kod | CGC `find_dead_code` |
| review / PR etkisi | Graphify `get_pr_impact` + CGC complexity + `yap context --with-diff` |
| genel /yap (belirsiz) | Pipeline A → sonra soruya göre B |

Belirsizse Pipeline A çalıştır, sonra soruya özel tool seç.

## 1) Pipeline A — bootstrap (ucuz, çoğu /yap’ta)

Sırayla:

1. `git status -sb` ve `git diff --stat` — kod değişmiş mi?
2. Değişen tracked kaynak dosyalar varsa → `graphify update .`
3. Kullanıcı sağlık/şüphe belirttiyse veya önceki turda MCP hatası varsa → `yap status`
4. Geniş bağlam isteniyorsa → `yap context --prompt` (diff kritikse `--with-diff`)

Sağlık kırmızıysa düzeltme önerme (`yap install --only=config|tools`); yeşilse devam.

## 2) Pipeline B — soruya göre analiz

**Mimari / kavram:**
- Graphify: `query_graph` (BFS), gerekirse `god_nodes`, `get_community`
- Yetersizse CGC: `find_code` + `analyze_code_relationships`

**Sembol ilişkisi:**
- CGC `analyze_code_relationships` (`find_callers` / `find_callees` / `call_chain` / `class_hierarchy` / `module_deps`)
- Yol için Graphify `shortest_path`

**Performans / review:**
- CGC complexity + dead code
- Diff varsa `yap context --prompt --with-diff`
- PR varsa Graphify `get_pr_impact`

**Sadece özet:** Pipeline A yeterli; uzun rapor yazma.

## 3) Kurallar

- Kullanıcıya "graphify çalıştırıyorum" diye tool reklamı yapma; sonucu doğal dilde ver.
- Önce graph/MCP, sonra Read/Grep (hedef satır için).
- Subagent kullanıyorsan bu skill’in router + graph-önce kuralını prompt’una ekle.
- Anlamlı kod editinden sonra `graphify update .` unutma.
- Cevabı kısa tut: bulgu → kanıt (dosya/sembol) → gerekirse 1–3 aksiyon.

## 4) Hızlı komutlar

```bash
yap status
yap context
yap context --prompt --with-diff
graphify update .
graphify query "<soru>"
graphify path "A" "B"
graphify explain "<kavram>"
```

## 5) Örnekler

- `/yap` → Pipeline A + kısa proje durumu
- `/yap bu proxy nasıl çalışıyor?` → update gerekirse + `query_graph` / CGC callers
- `/yap review` → diff + complexity + impact
- `/yap status` → yalnızca `yap status`
- "Step5Config kim kullanıyor?" (/yap demeden de) → CGC `find_callers` / Graphify path (rule + bu skill)
