/* ==========================================================================
   Galaxy Layout Engine
   Circular positioning for mood orbs. CSS handles drift animation.
   No physics loop — zero ongoing JS cost.
   ========================================================================== */

export class Galaxy {
  constructor() {
    this.orbs = [];
    this.containerWidth = 0;
    this.containerHeight = 0;
    this.container = null;
    // Exposed for app.js to conditionally enable galaxy mode CSS class
    this.reducedMotion = window.matchMedia('(prefers-reduced-motion: reduce)').matches;
  }

  /**
   * Initialize the galaxy with orb data.
   * @param {HTMLElement} container - The mood-galaxy element
   * @param {{el: HTMLElement, trackCount: number}[]} orbData
   */
  init(container, orbData) {
    this.container = container;
    this.updateBounds();

    if (orbData.length === 0) return;

    const baseSize = this._baseSize();

    this.orbs = orbData.map((data, i) => {
      data.el.style.setProperty('--orb-size', `${baseSize}px`);
      const radius = baseSize / 2;
      const pos = this._circularPosition(i, orbData.length, radius);
      return { el: data.el, x: pos.x, y: pos.y, radius };
    });

    this._applyPositions();
  }

  /** Update container bounds and reposition orbs on resize. */
  updateBounds() {
    if (!this.container) return;
    this.containerWidth = this.container.clientWidth;
    this.containerHeight = this.container.clientHeight;
    if (this.containerWidth < 1 || this.containerHeight < 1) return;
    this._relayout();
  }

  /** Recalculate sizes and positions. */
  _relayout() {
    if (this.orbs.length === 0) return;
    const n = this.orbs.length;
    const baseSize = this._baseSize();

    for (let i = 0; i < n; i++) {
      const orb = this.orbs[i];
      orb.radius = baseSize / 2;
      orb.el.style.setProperty('--orb-size', `${baseSize}px`);
      const pos = this._circularPosition(i, n, orb.radius);
      orb.x = pos.x;
      orb.y = pos.y;
    }
    this._applyPositions();
  }

  /** Apply orb positions to DOM via transform. */
  _applyPositions() {
    for (const orb of this.orbs) {
      const tx = orb.x - orb.radius;
      const ty = orb.y - orb.radius;
      orb.el.style.transform = `translate(${Math.round(tx)}px, ${Math.round(ty)}px)`;
    }
  }

  /** Base orb size scaled to container. */
  _baseSize() {
    let size;
    if (this.containerWidth >= 1024) size = 160;
    else if (this.containerWidth >= 768) size = 140;
    else size = Math.min(100, Math.max(80, Math.floor(this.containerWidth * 0.23)));

    // Height constraint: need enough room for the circular spread
    const maxFromHeight = Math.floor(this.containerHeight / 3.5);
    return Math.max(70, Math.min(size, maxFromHeight));
  }

  /** Circular position for orb i of n. Uses elliptical spread to fill container. */
  _circularPosition(i, n, radius) {
    const cx = this.containerWidth / 2;
    const cy = this.containerHeight / 2;
    const padding = radius * 1.2;

    if (n <= 1) return { x: cx, y: cy };

    // Elliptical spread — centered cluster with room between neighbours
    const angle = (i / n) * Math.PI * 2 - Math.PI / 2;
    const spreadX = Math.min(cx - padding, radius * 2.6);
    const spreadY = Math.min(cy - padding, radius * 2.2);

    return {
      x: Math.max(padding, Math.min(this.containerWidth - padding, cx + Math.cos(angle) * spreadX)),
      y: Math.max(padding, Math.min(this.containerHeight - padding, cy + Math.sin(angle) * spreadY))
    };
  }
}
