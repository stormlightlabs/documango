(function () {
  'use strict';

  const searchInput = document.querySelector('.search-input');
  const resultsContainer = document.querySelector('.results-list');
  const searchForm = document.querySelector('.search-form');
  const searchStats = document.querySelector('.search-stats');

  if (!searchInput || !searchForm) return;

  let debounceTimer;
  const DEBOUNCE_MS = 150;

  function debouncedSearch() {
    clearTimeout(debounceTimer);
    debounceTimer = setTimeout(performSearch, DEBOUNCE_MS);
  }

  async function performSearch() {
    const query = searchInput.value.trim();
    if (!query) return;

    const pkg = new URLSearchParams(window.location.search).get('pkg') || '';
    const params = new URLSearchParams({ q: query, limit: '20' });
    if (pkg) params.set('pkg', pkg);

    try {
      const response = await fetch(`/api/search?${params}`);
      if (!response.ok) throw new Error('Search failed');

      const data = await response.json();
      updateResults(data);
      updateURL(query, pkg);
    } catch (err) {
      console.error('Search error:', err);
    }
  }

  function updateResults(data) {
    if (!resultsContainer) return;

    if (data.results.length === 0) {
      resultsContainer.innerHTML = `
                <div class="search-empty">
                    <p>No results found for "${escapeHtml(data.query)}"</p>
                </div>
            `;
    } else {
      resultsContainer.innerHTML = data.results
        .map(
          (r) => `
                <article class="search-result card card-static">
                    <h2 class="search-result-title">
                        <a href="/doc/${escapeHtml(r.path)}">${escapeHtml(r.title)}</a>
                    </h2>
                    <p class="search-result-path">${escapeHtml(r.path)}</p>
                    <p class="search-result-snippet">${r.snippet}</p>
                    <div class="search-result-meta">
                        <span class="package-badge">${escapeHtml(r.package)}</span>
                        <span class="score-badge">Score: ${r.score.toFixed(2)}</span>
                    </div>
                </article>
            `
        )
        .join('');
    }

    if (searchStats) {
      const count = data.results.length;
      searchStats.textContent = `Found ${data.total} result${count !== 1 ? 's' : ''} for "${escapeHtml(data.query)}"`;
    }
  }

  function updateURL(query, pkg) {
    const params = new URLSearchParams();
    if (query) params.set('q', query);
    if (pkg) params.set('pkg', pkg);
    const newUrl = `${window.location.pathname}${params.toString() ? '?' + params.toString() : ''}`;
    window.history.replaceState({}, '', newUrl);
  }

  function escapeHtml(text) {
    const div = document.createElement('div');
    div.textContent = text;
    return div.innerHTML;
  }

  searchInput.addEventListener('input', debouncedSearch);

  searchForm.addEventListener('submit', function (e) {
    e.preventDefault();
    clearTimeout(debounceTimer);
    performSearch();
  });

  let focusedResultIndex = -1;
  let resultElements = [];

  function updateResultElements() {
    resultElements = resultsContainer ? resultsContainer.querySelectorAll('.search-result') : [];
  }

  function focusResult(index) {
    updateResultElements();
    if (resultElements.length === 0) return;

    if (focusedResultIndex >= 0 && focusedResultIndex < resultElements.length) {
      resultElements[focusedResultIndex].classList.remove('focused');
    }

    if (index < 0) {
      focusedResultIndex = resultElements.length - 1;
    } else if (index >= resultElements.length) {
      focusedResultIndex = 0;
    } else {
      focusedResultIndex = index;
    }

    const element = resultElements[focusedResultIndex];
    element.classList.add('focused');
    element.scrollIntoView({ behavior: 'smooth', block: 'nearest' });
  }

  function openFocusedResult() {
    updateResultElements();
    if (focusedResultIndex >= 0 && focusedResultIndex < resultElements.length) {
      const link = resultElements[focusedResultIndex].querySelector('a');
      if (link) {
        window.location.href = link.href;
      }
    }
  }

  document.addEventListener('keydown', function (e) {
    if (e.key === '/' && document.activeElement !== searchInput) {
      e.preventDefault();
      searchInput.focus();
      return;
    }

    if (e.key === 'Escape' && document.activeElement === searchInput) {
      searchInput.value = '';
      searchInput.blur();
      return;
    }

    if (document.activeElement !== searchInput) {
      if (e.key === 'j' || e.key === 'ArrowDown') {
        e.preventDefault();
        focusResult(focusedResultIndex + 1);
        return;
      }
      if (e.key === 'k' || e.key === 'ArrowUp') {
        e.preventDefault();
        focusResult(focusedResultIndex - 1);
        return;
      }
      if (e.key === 'Enter') {
        e.preventDefault();
        openFocusedResult();
        return;
      }
    }
  });

  const originalUpdateResults = updateResults;
  updateResults = function (data) {
    originalUpdateResults(data);
    focusedResultIndex = -1;
    updateResultElements();
  };

  updateResultElements();
})();
