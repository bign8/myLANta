console.log("script loaded");

function ready(fn) {
  if (document.attachEvent ? document.readyState === "complete" : document.readyState !== "loading"){
    fn();
  } else {
    document.addEventListener('DOMContentLoaded', fn);
  }
}

ready(function() {
  // document.forms[0].querySelector("[type=button]")
  var file = document.getElementById("file");
  document.getElementById("add").onclick = function() {
    file.click();
  };
  file.onchange = function(e) {
    file.parentNode.submit();
  };
});
