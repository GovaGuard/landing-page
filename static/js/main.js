// GovaGuard Landing Page - Interactive Enhancements

document.addEventListener('DOMContentLoaded', () => {
    // Smooth scroll for anchor links
    document.querySelectorAll('a[href^="#"]').forEach(anchor => {
        anchor.addEventListener('click', function (e) {
            e.preventDefault();
            const target = document.querySelector(this.getAttribute('href'));
            if (target) {
                target.scrollIntoView({
                    behavior: 'smooth',
                    block: 'start'
                });
            }
        });
    });

    // Intersection Observer for fade-in animations
    const observerOptions = {
        threshold: 0.1,
        rootMargin: '0px 0px -100px 0px'
    };

    const observer = new IntersectionObserver((entries) => {
        entries.forEach(entry => {
            if (entry.isIntersecting) {
                entry.target.style.opacity = '1';
                entry.target.style.transform = 'translateY(0)';
            }
        });
    }, observerOptions);

    // Observe service cards
    document.querySelectorAll('.service-card').forEach((card, index) => {
        card.style.opacity = '0';
        card.style.transform = 'translateY(30px)';
        card.style.transition = `all 0.6s ease ${index * 0.1}s`;
        observer.observe(card);
    });

    // Observe expertise items
    document.querySelectorAll('.expertise-item').forEach((item, index) => {
        item.style.opacity = '0';
        item.style.transform = 'translateY(30px)';
        item.style.transition = `all 0.6s ease ${index * 0.1}s`;
        observer.observe(item);
    });

    // Terminal typewriter effect
    const terminalLines = document.querySelectorAll('.terminal-line');
    terminalLines.forEach((line, index) => {
        line.style.opacity = '0';
        setTimeout(() => {
            line.style.transition = 'opacity 0.3s ease';
            line.style.opacity = '1';
        }, index * 400);
    });

    // Add glitch effect to logo on hover
    const logo = document.querySelector('.logo');
    if (logo) {
        logo.addEventListener('mouseenter', () => {
            logo.style.animation = 'glitch 0.3s ease';
        });

        logo.addEventListener('animationend', () => {
            logo.style.animation = '';
        });
    }

    // Parallax effect for hero grid
    const gridOverlay = document.querySelector('.grid-overlay');
    if (gridOverlay) {
        window.addEventListener('scroll', () => {
            const scrolled = window.pageYOffset;
            gridOverlay.style.transform = `translateY(${scrolled * 0.3}px)`;
        });
    }

    // Form validation enhancement
    const contactForm = document.getElementById('contact-form');
    if (contactForm) {
        const inputs = contactForm.querySelectorAll('input, textarea, select');

        inputs.forEach(input => {
            input.addEventListener('blur', () => {
                if (input.value.trim() !== '') {
                    input.style.borderColor = 'var(--color-accent)';
                } else if (input.hasAttribute('required')) {
                    input.style.borderColor = 'var(--color-border)';
                }
            });

            input.addEventListener('focus', () => {
                input.style.borderColor = 'var(--color-primary)';
            });
        });
    }

    // Add active state to nav links based on scroll position
    const sections = document.querySelectorAll('section[id]');
    const navLinks = document.querySelectorAll('.nav-link');

    window.addEventListener('scroll', () => {
        let current = '';

        sections.forEach(section => {
            const sectionTop = section.offsetTop;
            const sectionHeight = section.clientHeight;
            if (pageYOffset >= sectionTop - 200) {
                current = section.getAttribute('id');
            }
        });

        navLinks.forEach(link => {
            link.style.color = 'var(--color-text-dim)';
            if (link.getAttribute('href') === `#${current}`) {
                link.style.color = 'var(--color-primary)';
            }
        });
    });

    // Animate stats on scroll
    const stats = document.querySelectorAll('.stat-number');
    let hasAnimated = false;

    const statsObserver = new IntersectionObserver((entries) => {
        entries.forEach(entry => {
            if (entry.isIntersecting && !hasAnimated) {
                hasAnimated = true;
                animateStats();
            }
        });
    }, { threshold: 0.5 });

    if (stats.length > 0) {
        statsObserver.observe(stats[0].closest('.hero-stats'));
    }

    function animateStats() {
        stats.forEach(stat => {
            const text = stat.textContent;
            if (text.includes('%')) {
                animateNumber(stat, 0, 98, 2000, '%');
            }
        });
    }

    function animateNumber(element, start, end, duration, suffix = '') {
        let startTime = null;

        function animation(currentTime) {
            if (!startTime) startTime = currentTime;
            const progress = Math.min((currentTime - startTime) / duration, 1);
            const value = Math.floor(progress * (end - start) + start);
            element.textContent = value + suffix;

            if (progress < 1) {
                requestAnimationFrame(animation);
            }
        }

        requestAnimationFrame(animation);
    }
});

// Add CSS for glitch animation
const style = document.createElement('style');
style.textContent = `
    @keyframes glitch {
        0% {
            transform: translate(0);
        }
        20% {
            transform: translate(-2px, 2px);
        }
        40% {
            transform: translate(-2px, -2px);
        }
        60% {
            transform: translate(2px, 2px);
        }
        80% {
            transform: translate(2px, -2px);
        }
        100% {
            transform: translate(0);
        }
    }
`;
document.head.appendChild(style);
