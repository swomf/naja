let activeCard = null;

// click handler since use case is simple
// basically the same as adding a listener to everything and
// only filtering out what has a data-action
document.body.addEventListener("click", (e) => {
  const t = e.target;
  const card = t.closest(".card");
  if (card) activeCard = card;

  const action = t.dataset.action;
  if (!action) return;

  const id = t.dataset.id;
  const src = t.dataset.src;
  if (!id || !src) {
    // future proofing in case i add more data-actions
    console.warn("missing data-id or data-src on", t);
    return;
  }

  if (action === "load") loadVideo(id, src);
  else if (action === "switch") switchVideo(id, src);
  // else console.warn("invalid data-action " + t.dataset.action)
});

// kind of dirty but keyboard switch applies to last card interacted with
document.addEventListener("keydown", (e) => {
  const target = e.target;
  if (target && typeof target.matches === "function" &&
    target.matches("input, textarea, select, [contenteditable=\"true\"]")) {
    return;
  }

  let direction;
  if (e.key === "[" || e.code === "BracketLeft") direction = -1;
  else if (e.key === "]" || e.code === "BracketRight" ||
    e.key === "s" || e.key === "S" || e.code === "KeyS") direction = 1;
  else return;

  const card = activeCard || document.querySelector(".card");
  if (!card) return;

  e.preventDefault();
  cycleVideo(card, direction);
}, { capture: true });

function loadVideo(id, src) {
  const c = document.getElementById(id + "-container");
  if (!c) return;
  c.innerHTML = `
    <video id="${id}-player" controls autoplay width="100%">
      <source src="${src}" type="video/mp4">
    </video>
  `;
  setCurrentVideo(c.closest(".card"), src);
}

function setCurrentVideo(card, src) {
  if (!card) return;
  const buttons = [...card.querySelectorAll('[data-action="switch"]')];
  const index = buttons.findIndex((button) => button.dataset.src === src);
  if (index !== -1) card.dataset.currentIndex = index;
}

function cycleVideo(card, direction) {
  const buttons = [...card.querySelectorAll('[data-action="switch"]')];
  if (buttons.length < 2) return;

  let index = Number.parseInt(card.dataset.currentIndex, 10);
  if (!Number.isInteger(index)) index = 0;
  index = (index + direction + buttons.length) % buttons.length;
  switchVideo(card.dataset.id, buttons[index].dataset.src);
}

function switchVideo(id, src) {
  const player = document.getElementById(id + "-player");
  const container = document.getElementById(id + "-container");
  if (!container) return;
  setCurrentVideo(container.closest(".card"), src);

  // height lock so that switching a video doesn't
  // cause disorienting squeeze when video is replaced
  const currentHeight = container.offsetHeight;
  container.style.minHeight = currentHeight + "px";

  if (!player) {
    // switch will get called before load if the bottom
    // buttons are pressed first, before the thumbnail...
    // must load video player before switches happen
    loadVideo(id, src);
    container.style.minHeight = "";
    return;
  }
  const wasPlaying = !player.paused;
  const t = player.currentTime;
  player.src = src;
  player.load();
  player.currentTime = t;
  if (wasPlaying) player.play();

  player.onloadedmetadata = () => {
    container.style.minHeight = "";
  };
}
