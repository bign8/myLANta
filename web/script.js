function ready(fn) {
  if (document.attachEvent ? document.readyState === "complete" : document.readyState !== "loading"){
    fn();
  } else {
    document.addEventListener('DOMContentLoaded', fn);
  }
}

ready(function() {
  var form = document.forms[0],
    save = form.querySelector("[type=submit]"),
    file = form.querySelector("[type=file]"),
    best = document.createElement('input');

  best.type = 'button';
  best.value = save.value;
  file.style.display = 'none';
  save.style.display = 'none';

  form.appendChild(best);
  best.onclick = function(e) { file.click(e); };
  file.onchange = function(e) { form.submit(e); };
});
