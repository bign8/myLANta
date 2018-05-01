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

    // If we have JS -- replace the form post to be a JS handler.
    // Make the chat window also handled by JS instead of an iframe.
    var form = document.getElementById('chatform');
    if (form.attachEvent) {
        form.attachEvent("submit", processForm);
    } else {
        form.addEventListener("submit", processForm);
    }
    var pane = document.getElementById('chatpane');
    var chatsys = pane.parentNode;
    chatsys.removeChild(pane);
    var pane = document.createElement('div');
    pane.className = "chatdiv"
    chatsys.insertBefore(pane, form);
    getChat(pane);
  }

  function processForm(e) {
    if (e.preventDefault) e.preventDefault();
    postChat(e)
    // You must return false to prevent the default form behavior
    return false;
  }

  function getChat(pane) {
    var request = new XMLHttpRequest();
    request.onload = function () {
      pane.innerHTML = request.responseText;
      pane.scrollTo(0, pane.scrollHeight);
      window.setTimeout(getChat(pane), 250);
    }
    request.open("GET", "/chat", true);
    request.send()
  }

  function postChat(formData) {
    var request = new XMLHttpRequest();
    request.onload = function () {
    }
    request.open("POST", "/chat", true);
    request.setRequestHeader("Content-Type", "application/json");
    request.send(JSON.stringify({
      "who": formData.target[0].value,
      "msg": formData.target[1].value
    }));
  }

  // When the document is ready, run some initialization scrips.
  ready(function() {
    initFile('file');
    initChat('chat');
  });
})(document);
