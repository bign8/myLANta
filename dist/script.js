(function(d) {
  function ready(fn) {
    if (d.attachEvent ? d.readyState === "complete" : d.readyState !== "loading"){
      fn();
    } else {
      d.addEventListener('DOMContentLoaded', fn);
    }
  }

  // Initialize the file upload form.
  function initFile(name) {
    var form = d.forms[name],
      save = form.querySelector("[type=submit]"),
      file = form.querySelector("[type=file]"),
      best = d.createElement('input');

    best.type = 'button';
    best.value = save.value;
    file.style.display = 'none';
    save.style.display = 'none';

    form.appendChild(best);
    best.onclick = function(e) { file.click(e); };
    file.onchange = function(e) { form.submit(e); };
  }

  // Initialize the chat form.
  function initChat(name) {
    var form = d.forms[name],
      send = form.querySelector("[type=submit]"),
      name = form.querySelector("[name=who]"),
      text = form.querySelector("[name=msg]");

    name.value = window.localStorage["name"];

    send.onclick = function(e) {
      window.localStorage["name"] = name.value; // save username

      // wait till after form submits to clear text field
      window.setTimeout(function() {
        text.value = '';
      }, 1);
    };
  }

  // When the document is ready, run some initialization scrips.
  ready(function() {
    initFile('file');
    initChat('chat');
  });
})(document);
