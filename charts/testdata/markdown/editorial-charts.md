# Editorial charts

```goshtosochart
schemaVersion: 1
type: bar
title: Revenue
categories: [Q1, Q2]
series: [{name: Revenue, values: [10, 12]}]
```

```goshtosochart
schemaVersion: 1
type: line
title: Trend
categories: [Q1, Q2]
series: [{name: Revenue, values: [10, 12]}]
```

```goshtosochart
schemaVersion: 1
type: pie
title: Mix
slices: [{name: Desktop, value: 40}, {name: Mobile, value: 60}]
```

```goshtosochart
schemaVersion: 1
type: scatter
title: Latency
categories: [p50, p95]
series: [{name: API, points: [{category: p50, value: 12}, {category: p95, value: 18}]}]
```
