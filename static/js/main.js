(function () {
  'use strict';

  // ---------- Шапка: тень при прокрутке ----------
  var header = document.querySelector('.header');
  function onScroll() {
    header.classList.toggle('is-scrolled', window.scrollY > 8);
  }
  window.addEventListener('scroll', onScroll, { passive: true });
  onScroll();

  // ---------- Мобильное меню ----------
  var burger = document.getElementById('burger');
  var nav = document.getElementById('nav');
  function setMenu(open) {
    nav.classList.toggle('is-open', open);
    burger.setAttribute('aria-expanded', String(open));
    burger.setAttribute('aria-label', open ? 'Закрыть меню' : 'Открыть меню');
  }
  burger.addEventListener('click', function () {
    setMenu(!nav.classList.contains('is-open'));
  });
  nav.addEventListener('click', function (e) {
    if (e.target.closest('a')) setMenu(false);
  });
  document.addEventListener('keydown', function (e) {
    if (e.key === 'Escape') setMenu(false);
  });

  // ---------- Появление блоков при прокрутке ----------
  var revealEls = document.querySelectorAll('.reveal');
  if ('IntersectionObserver' in window) {
    var io = new IntersectionObserver(function (entries) {
      entries.forEach(function (entry) {
        if (!entry.isIntersecting) return;
        // Лёгкая «лесенка» для соседних карточек
        var siblings = Array.prototype.indexOf.call(entry.target.parentNode.children, entry.target);
        entry.target.style.transitionDelay = Math.min(siblings, 5) * 70 + 'ms';
        entry.target.classList.add('is-visible');
        io.unobserve(entry.target);
      });
    }, { rootMargin: '0px 0px -8% 0px', threshold: 0.08 });
    revealEls.forEach(function (el) { io.observe(el); });
  } else {
    revealEls.forEach(function (el) { el.classList.add('is-visible'); });
  }

  // ---------- Кнопки тарифов подставляют тариф в форму ----------
  var packageSelect = document.getElementById('package');
  document.querySelectorAll('[data-plan]').forEach(function (btn) {
    btn.addEventListener('click', function () {
      packageSelect.value = btn.getAttribute('data-plan');
    });
  });

  // ---------- Маска телефона +7 (XXX) XXX-XX-XX ----------
  var phoneInput = document.querySelector('input[name="phone"]');
  phoneInput.addEventListener('input', function () {
    var digits = phoneInput.value.replace(/\D/g, '');
    if (!digits) { phoneInput.value = ''; return; }
    if (digits[0] === '8') digits = '7' + digits.slice(1);
    if (digits[0] !== '7') digits = '7' + digits;
    digits = digits.slice(0, 11);
    var d = digits.slice(1);
    var out = '+7';
    if (d.length) out += ' (' + d.slice(0, 3);
    if (d.length >= 3) out += ')';
    if (d.length > 3) out += ' ' + d.slice(3, 6);
    if (d.length > 6) out += '-' + d.slice(6, 8);
    if (d.length > 8) out += '-' + d.slice(8, 10);
    phoneInput.value = out;
  });

  // ---------- Отправка заявки ----------
  var form = document.getElementById('lead-form');
  var status = form.querySelector('.form__status');
  var submitBtn = form.querySelector('button[type="submit"]');

  function setStatus(text, type) {
    status.textContent = text;
    status.className = 'form__status' + (type ? ' is-' + type : '');
  }

  function markInvalid(input, invalid) {
    input.closest('.field').classList.toggle('is-invalid', invalid);
  }

  form.addEventListener('submit', function (e) {
    e.preventDefault();

    var data = Object.fromEntries(new FormData(form).entries());
    var nameInput = form.elements.name;
    var nameOk = data.name.trim().length > 0;
    var phoneOk = data.phone.replace(/\D/g, '').length >= 11;
    markInvalid(nameInput, !nameOk);
    markInvalid(phoneInput, !phoneOk);
    if (!nameOk || !phoneOk) {
      setStatus('Заполните имя и полный номер телефона.', 'error');
      (nameOk ? phoneInput : nameInput).focus();
      return;
    }

    submitBtn.disabled = true;
    setStatus('Отправляем…');

    fetch('/api/lead', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(data)
    })
      .then(function (res) { return res.json().catch(function () { return {}; }); })
      .then(function (json) {
        if (json.ok) {
          form.reset();
          setStatus('Спасибо! Заявка отправлена, скоро свяжусь с вами.', 'ok');
        } else {
          setStatus(json.error || 'Что-то пошло не так. Напишите мне в WhatsApp.', 'error');
        }
      })
      .catch(function () {
        setStatus('Нет соединения. Проверьте интернет или напишите в WhatsApp.', 'error');
      })
      .finally(function () { submitBtn.disabled = false; });
  });

  form.addEventListener('input', function (e) {
    var field = e.target.closest('.field');
    if (field) field.classList.remove('is-invalid');
  });
})();
