'use client';

import Link from 'next/link';
import Image from 'next/image';
import React from 'react';
import { Menu, X } from 'lucide-react';
import { Button } from '@/components/ui/button';
import { cn } from '@/lib/utils';
import { ThemeToggle } from './theme-toggle';

const menuItems = [
  { name: 'Beranda', href: '/' },
  { name: 'Siswa', href: '/users' },
  { name: 'Portofolio', href: '/portfolios' },
  { name: 'Changelog', href: '/changelog' },
];

export function GuestNavbar() {
  const [menuState, setMenuState] = React.useState(false);
  const [isScrolled, setIsScrolled] = React.useState(false);

  React.useEffect(() => {
    const handleScroll = () => {
      setIsScrolled(window.scrollY > 50);
    };
    window.addEventListener('scroll', handleScroll);
    return () => window.removeEventListener('scroll', handleScroll);
  }, []);

  return (
    <header>
      <nav className={cn('fixed z-20 w-full px-2 transition-all duration-300', isScrolled ? 'top-2' : 'top-0')}>
        <div
          className={cn(
            'mx-auto max-w-6xl rounded-2xl px-6 backdrop-blur-md transition-all duration-300 lg:px-12 relative z-50',
            isScrolled
              ? 'max-w-4xl border bg-background/80 shadow-sm lg:px-5'
              : 'bg-background/0 mt-4'
          )}
        >
          <div className="relative flex items-center justify-between py-3 lg:gap-0 lg:py-4">
            {/* Logo */}
            <Link href="/" aria-label="home" className="flex items-center space-x-2 z-50">
              <Image
                src="/images/logos/logo_black.svg"
                alt="Grafikarsa"
                width={28}
                height={28}
                className="block dark:hidden"
              />
              <Image
                src="/images/logos/logo_white.svg"
                alt="Grafikarsa"
                width={28}
                height={28}
                className="hidden dark:block"
              />
              <span className="text-xl font-bold text-foreground">Grafikarsa</span>
            </Link>

            {/* Mobile Menu Button */}
            <button
              onClick={() => setMenuState(!menuState)}
              aria-label={menuState ? 'Close Menu' : 'Open Menu'}
              className="relative z-50 -m-2.5 block cursor-pointer p-2.5 lg:hidden"
            >
              <Menu className={cn("size-6 duration-200", menuState ? "hidden" : "block")} />
              <X className={cn("size-6 duration-200", menuState ? "block" : "hidden")} />
            </button>

            {/* Desktop Navigation */}
            <div className="absolute inset-0 m-auto hidden size-fit lg:block">
              <ul className="flex gap-8 text-sm">
                {menuItems.map((item, index) => (
                  <li key={index}>
                    <Link
                      href={item.href}
                      className="block text-foreground/80 duration-150 hover:text-foreground"
                    >
                      <span>{item.name}</span>
                    </Link>
                  </li>
                ))}
              </ul>
            </div>

            {/* Desktop Actions */}
            <div className="hidden lg:flex lg:items-center lg:gap-3">
              <ThemeToggle />
              <Button asChild variant={isScrolled ? "default" : "outline"} size="sm">
                <Link href="/login">
                  <span>Login</span>
                </Link>
              </Button>
            </div>
          </div>
        </div>

        {/* Mobile Menu Overlay */}
        <div
          className={cn(
            "fixed inset-x-0 top-0 z-40 flex flex-col gap-4 bg-background px-6 pb-6 pt-24 shadow-2xl transition-all duration-300 ease-in-out lg:hidden",
            menuState ? "translate-y-0 opacity-100" : "-translate-y-full opacity-0"
          )}
        >
          <ul className="flex flex-col gap-4 text-base">
            {menuItems.map((item, index) => (
              <li key={index}>
                <Link
                  href={item.href}
                  className="block py-2 text-lg font-medium text-foreground/80 hover:text-foreground"
                  onClick={() => setMenuState(false)}
                >
                  <span>{item.name}</span>
                </Link>
              </li>
            ))}
          </ul>
          <div className="mt-4 flex flex-col gap-3">
            <div className="flex items-center justify-between border-t pt-4">
              <span className="text-sm font-medium">Tema</span>
              <ThemeToggle />
            </div>
            <Button asChild size="lg" className="w-full">
              <Link href="/login" onClick={() => setMenuState(false)}>
                <span>Login</span>
              </Link>
            </Button>
          </div>
        </div>
      </nav>
    </header >
  );
}
