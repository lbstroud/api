function scroll() {
  var fragment = (new URL(document.location.href)).hash

  if (fragment != "") {
    fragment = fragment.slice(1); // remove # prefix

    var spec = document.getElementById("spec");
    spec.addEventListener('spec-loaded', function(e) {
      console.log("scrolling to " + fragment);
      setTimeout(function() {
        spec.scrollTo(fragment);
      }, 1000);
    });
  }
}
