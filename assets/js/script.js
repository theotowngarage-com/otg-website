addEventListener("DOMContentLoaded", (event) => {
  var navbar = document.querySelector('.site-navigation');
  addEventListener("scroll", (event) => {
    if (window.scrollY >= 100) {
      navbar.classList.add('nav-bg');
    } else {
      navbar.classList.remove('nav-bg');
    }
  });
});

