/*
Copyright (C) 2025 Xauryan

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as
published by the Free Software Foundation, either version 3 of the
License, or (at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
GNU Affero General Public License for more details.

You should have received a copy of the GNU Affero General Public License
along with this program. If not, see <https://www.gnu.org/licenses/>.

For commercial licensing, please contact support@xauryan.com
*/

import React, { useEffect, useRef } from 'react';
import './quantumHome.css';
import Hero from './components/Hero';
import HowItWorks from './components/HowItWorks';
import FeatureGrid from './components/FeatureGrid';
import ProviderConstellation from './components/ProviderConstellation';
import TerminalDemo from './components/TerminalDemo';
import CTA from './components/CTA';

const QuantumHome = (props) => {
  const rootRef = useRef(null);

  useEffect(() => {
    const root = rootRef.current;
    if (!root) return undefined;
    const reveals = Array.from(root.querySelectorAll('.sh-reveal'));
    if (reveals.length === 0) return undefined;

    if (typeof IntersectionObserver === 'undefined') {
      reveals.forEach((el) => el.classList.add('is-visible'));
      return undefined;
    }

    const observer = new IntersectionObserver(
      (entries) => {
        entries.forEach((entry) => {
          if (entry.isIntersecting) {
            entry.target.classList.add('is-visible');
            observer.unobserve(entry.target);
          }
        });
      },
      { threshold: 0.12, rootMargin: '0px 0px -10% 0px' },
    );

    reveals.forEach((el) => observer.observe(el));
    return () => observer.disconnect();
  }, []);

  return (
    <div className='sh-home' ref={rootRef}>
      <div className='sh-bg' aria-hidden='true'>
        <div className='sh-bg-dots' />
        <div className='sh-bg-mesh sh-bg-mesh-1' />
        <div className='sh-bg-mesh sh-bg-mesh-2' />
        <div className='sh-bg-divider' />
      </div>
      <Hero {...props} />
      <HowItWorks {...props} />
      <FeatureGrid {...props} />
      <ProviderConstellation {...props} />
      <TerminalDemo {...props} />
      <CTA {...props} />
    </div>
  );
};

export default QuantumHome;
