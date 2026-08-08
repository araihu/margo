## 13. Static chart projections

Charts are static Goshtoso SVG projections over the same Markdown document.
The benchmark covers the four v1 families, their accessible data tables, the
theme-token default, a caller class, and explicit hexadecimal colors.

### Revenue by team

```goshtosochart
schemaVersion: 1
type: bar
title: Revenue by team
style:
  palette: pastel
  class: benchmark-revenue-chart
categories: [Development, Production]
series:
  - name: Revenue
    class: benchmark-revenue-series
    values: [12, 18]
  - name: Cost
    color: "#dc2626"
    values: [7, 9]
```

### Quarterly trend

```goshtosochart
schemaVersion: 1
type: line
title: Quarterly trend
style:
  class: benchmark-trend-chart
categories: [Q1, Q2, Q3, Q4]
series:
  - name: Revenue
    color: "#2563eb"
    values: [12, 15, 18, 22]
  - name: Cost
    class: benchmark-cost-series
    values: [7, 9, 11, 13]
```

### Traffic mix

```goshtosochart
schemaVersion: 1
type: doughnut
title: Traffic mix
style:
  palette: status
  class: benchmark-traffic-chart
slices:
  - name: Desktop
    class: benchmark-desktop-slice
    value: 40
  - name: Mobile
    color: "#0f766e"
    value: 60
```

### Latency distribution

```goshtosochart
schemaVersion: 1
type: scatter
title: Latency distribution
style:
  palette: neutral
  class: benchmark-latency-chart
categories: [p50, p95]
series:
  - name: API
    class: benchmark-api-series
    points:
      - category: p50
        value: 12
      - category: p95
        value: 18
  - name: Worker
    color: "#7c3aed"
    points:
      - category: p50
        value: 30
      - category: p95
        value: 42
```
