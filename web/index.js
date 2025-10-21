// click handler since use case is simple
// basically the same as adding a listener to everything and
// only filtering out what has a data-action
document.body.addEventListener("click", (e) => {
  const t = e.target;
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

function loadVideo(id, src) {
  const c = document.getElementById(id + "-container");
  c.innerHTML = `
    <video id="${id}-player" controls autoplay width="100%">
      <source src="${src}" type="video/mp4">
    </video>
  `;
}

function switchVideo(id, src) {
  const player = document.getElementById(id + "-player");
  const container = document.getElementById(id + "-container");
  if (!container) return;

  // height lock so that switching a video doesn't
  // cause disorienting squeeze when video is replaced
  const currentHeight = container.offsetHeight;
  container.style.minHeight = currentHeight + "px";

  if (!player) {
    // switch will get called before load if the bottom
    // buttons are pressed first, before the thumbnail...
    // must load video player before switches happen
    loadVideo(id, src);
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
