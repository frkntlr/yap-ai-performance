---
name: yap
description: >-
  Proje genelinde CodeGraphContext ve Graphify MCP sunucularını kullanarak
  derinlemesine analiz, optimizasyon ve kod incelemesi yapar. Kullanıcı /yap,
  yap veya proje analizi istediğinde çağırır.
---

# Yap AI Performance Analyzer

Bu beceri çağrıldığında CodeGraphContext ve Graphify MCP araçlarını kullanarak
proje yapısını, ilişkilerini ve karmaşıklığını analiz et.

## Görevler

1. Bağlı MCP tool'larıyla kod grafiği / çağrı ilişkisi / complexity analizi yap.
2. /yap veya "yap" denince: gerekirse graph güncelle, sağlık kontrolü, kısa mimari özet.
3. Önce MCP/graph verisi, sonra dosya okuma.

## Kurallar

- Tool adını reklam etme; bulguları doğal dilde ver.
- Anlamlı kod değişiminden sonra `graphify update .` öner veya çalıştır.
