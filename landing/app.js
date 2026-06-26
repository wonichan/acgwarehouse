(function() {
  "use strict";
  try {
    var chips = document.querySelectorAll('.chip');
    var cards = document.querySelectorAll('.gallery-card');
    chips.forEach(function(chip) {
      chip.addEventListener('click', function() {
        var filter = chip.getAttribute('data-filter');
        chips.forEach(function(c) { c.classList.remove('is-active'); c.setAttribute('aria-pressed','false'); });
        chip.classList.add('is-active');
        chip.setAttribute('aria-pressed','true');
        cards.forEach(function(card) {
          var cat = card.getAttribute('data-category');
          if (filter === 'all' || cat === filter) {
            card.classList.remove('is-hidden');
          } else {
            card.classList.add('is-hidden');
          }
        });
      });
    });

    var lightbox = document.getElementById('lightbox');
    var lightboxArt = document.getElementById('lightbox-art');
    var lightboxTitle = document.getElementById('lightbox-title');
    var closeBtn = lightbox.querySelector('.lightbox-close');

    function openLightbox(card) {
      var art = card.querySelector('.card-art');
      var title = card.querySelector('.card-title').textContent;
      lightboxArt.innerHTML = '';
      var clone = art.cloneNode(true);
      clone.style.aspectRatio = '16/9';
      lightboxArt.appendChild(clone);
      lightboxTitle.textContent = title;
      lightbox.classList.add('is-open');
      lightbox.setAttribute('aria-hidden','false');
      document.body.style.overflow = 'hidden';
    }

    function closeLightbox() {
      lightbox.classList.remove('is-open');
      lightbox.setAttribute('aria-hidden','true');
      document.body.style.overflow = '';
    }

    cards.forEach(function(card) {
      card.addEventListener('click', function() { openLightbox(card); });
    });

    closeBtn.addEventListener('click', function(e) { e.stopPropagation(); closeLightbox(); });
    lightbox.querySelector('.lightbox-backdrop').addEventListener('click', closeLightbox);

    document.addEventListener('keydown', function(e) {
      if (e.key === 'Escape' && lightbox.classList.contains('is-open')) {
        closeLightbox();
      }
    });
  } catch (e) {
    if (typeof console !== 'undefined' && console.warn) {
      console.warn('ACG Landing enhancement failed silently:', e);
    }
  }
})();
