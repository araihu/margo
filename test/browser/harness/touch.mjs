export async function moveTouches(page, session, from, to, steps = 8) {
  for (let step = 1; step <= steps; step++) {
    const fraction = step / steps;
    await session.send("Input.dispatchTouchEvent", {type: "touchMove", touchPoints: from.map((point, index) => ({
      id: point.id,
      x: point.x + (to[index].x - point.x) * fraction,
      y: point.y + (to[index].y - point.y) * fraction
    }))});
    // Deliver separate frames, as a physical gesture does, so the browser can
    // recognize native page zoom as well as the viewer's pointer handling.
    await page.waitForTimeout(16);
  }
}

export async function pinch(page, session, center, ratio, offset = {x: 0, y: 0}, end = true) {
  const from = [{id: 1, x: center.x - 30, y: center.y}, {id: 2, x: center.x + 30, y: center.y}];
  const to = from.map((point) => ({id: point.id, x: center.x + (point.x - center.x) * ratio + offset.x, y: point.y + offset.y}));
  await session.send("Input.dispatchTouchEvent", {type: "touchStart", touchPoints: from});
  await moveTouches(page, session, from, to);
  if (end) await session.send("Input.dispatchTouchEvent", {type: "touchEnd", touchPoints: []});
  return to;
}

