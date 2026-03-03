'use client';

import { useEffect, useState } from 'react';
import Image from 'next/image';
import { Heart } from 'lucide-react';
import { Avatar, AvatarFallback } from '@/components/ui/avatar';
import { Badge } from '@/components/ui/badge';
import { Card } from '@/components/ui/card';
import { Button } from '@/components/ui/button';
import { cn } from '@/lib/utils';

// Portfolio cards fan stack animation with images
export function PortfolioStack() {
  const [activeIndex, setActiveIndex] = useState(0);
  
  const cards = [
    { 
      title: 'Website Portfolio', 
      tag: 'Web Dev', 
      user: 'Andi',
      image: 'https://images.unsplash.com/photo-1460925895917-afdab827c52f?w=400&h=300&fit=crop'
    },
    { 
      title: 'Brand Identity', 
      tag: 'Design', 
      user: 'Budi',
      image: 'https://images.unsplash.com/photo-1561070791-2526d30994b5?w=400&h=300&fit=crop'
    },
    { 
      title: 'Mobile App UI', 
      tag: 'UI/UX', 
      user: 'Citra',
      image: 'https://images.unsplash.com/photo-1512941937669-90a1b58e7e9c?w=400&h=300&fit=crop'
    },
  ];

  useEffect(() => {
    const interval = setInterval(() => {
      setActiveIndex((prev) => (prev + 1) % cards.length);
    }, 3000);
    return () => clearInterval(interval);
  }, [cards.length]);

  return (
    <div className="relative h-full w-full flex items-center justify-center">
      {cards.map((card, i) => {
        const offset = (i - activeIndex + cards.length) % cards.length;
        // Fan spread: rotate and translate like a card fan
        const rotations = [-15, 0, 15];
        const translateX = [-40, 0, 40];
        const translateY = [20, 0, 20];
        const zIndexes = [10, 30, 10];
        const scales = [0.9, 1, 0.9];
        
        return (
          <Card
            key={i}
            className="absolute w-48 transition-all duration-700 ease-out p-0 overflow-hidden shadow-lg"
            style={{
              transform: `rotate(${rotations[offset]}deg) translateX(${translateX[offset]}px) translateY(${translateY[offset]}px) scale(${scales[offset]})`,
              zIndex: zIndexes[offset],
            }}
          >
            <div className="relative aspect-[4/3] w-full overflow-hidden">
              <Image
                src={card.image}
                alt={card.title}
                fill
                className="object-cover"
              />
            </div>
            <div className="p-3">
              <Badge variant="secondary" className="text-[10px] px-1.5 py-0 mb-1.5">
                {card.tag}
              </Badge>
              <p className="text-xs font-medium truncate">{card.title}</p>
              <div className="flex items-center gap-1.5 mt-2">
                <Avatar className="h-5 w-5">
                  <AvatarFallback className="text-[9px]">{card.user[0]}</AvatarFallback>
                </Avatar>
                <span className="text-[10px] text-muted-foreground">{card.user}</span>
              </div>
            </div>
          </Card>
        );
      })}
    </div>
  );
}

// Floating avatars with follow animation
export function FloatingAvatars() {
  const [followedIndex, setFollowedIndex] = useState<number | null>(null);
  
  const users = [
    { name: 'Andi', initial: 'A' },
    { name: 'Budi', initial: 'B' },
    { name: 'Citra', initial: 'C' },
    { name: 'Dewi', initial: 'D' },
  ];

  useEffect(() => {
    const interval = setInterval(() => {
      setFollowedIndex((prev) => {
        if (prev === null) return 0;
        if (prev >= users.length - 1) return null;
        return prev + 1;
      });
    }, 1500);
    return () => clearInterval(interval);
  }, [users.length]);

  return (
    <div className="h-full w-full p-3 flex flex-col justify-center">
      <div className="space-y-1.5">
        {users.map((user, i) => (
          <div
            key={i}
            className={cn(
              'flex items-center justify-between rounded-lg border bg-card p-2 transition-all duration-300',
              followedIndex === i && 'border-primary/50 shadow-sm'
            )}
          >
            <div className="flex items-center gap-2">
              <Avatar className="h-6 w-6">
                <AvatarFallback className="text-[10px]">{user.initial}</AvatarFallback>
              </Avatar>
              <div>
                <p className="text-[11px] font-medium leading-tight">{user.name}</p>
                <p className="text-[9px] text-muted-foreground">@{user.name.toLowerCase()}</p>
              </div>
            </div>
            <Button
              size="sm"
              variant={followedIndex !== null && i <= followedIndex ? 'secondary' : 'default'}
              className="h-5 text-[9px] px-2"
            >
              {followedIndex !== null && i <= followedIndex ? 'Following' : 'Follow'}
            </Button>
          </div>
        ))}
      </div>
    </div>
  );
}

// Feed vertical marquee animation
export function FeedScrollAnimation() {
  const items = [
    { user: 'Andi', action: 'posted new portfolio', time: '2m' },
    { user: 'Budi', action: 'liked your work', time: '5m' },
    { user: 'Citra', action: 'started following you', time: '10m' },
    { user: 'Dewi', action: 'posted new portfolio', time: '15m' },
    { user: 'Eka', action: 'liked your work', time: '20m' },
    { user: 'Fajar', action: 'started following you', time: '25m' },
  ];

  // Duplicate items for seamless loop
  const allItems = [...items, ...items];

  return (
    <div className="h-full w-full overflow-hidden relative">
      {/* Fade masks */}
      <div className="absolute inset-x-0 top-0 h-8 bg-gradient-to-b from-card to-transparent z-10" />
      <div className="absolute inset-x-0 bottom-0 h-8 bg-gradient-to-t from-card to-transparent z-10" />
      
      {/* Marquee container */}
      <div className="animate-marquee-vertical space-y-1.5 p-2.5">
        {allItems.map((item, i) => (
          <div
            key={`${item.user}-${i}`}
            className="flex items-start gap-2 rounded-lg border bg-card p-2"
          >
            <Avatar className="h-5 w-5 shrink-0">
              <AvatarFallback className="text-[9px]">{item.user[0]}</AvatarFallback>
            </Avatar>
            <div className="min-w-0 flex-1">
              <p className="text-[10px] leading-tight">
                <span className="font-medium">{item.user}</span>{' '}
                <span className="text-muted-foreground">{item.action}</span>
              </p>
              <p className="text-[9px] text-muted-foreground">{item.time}</p>
            </div>
          </div>
        ))}
      </div>
    </div>
  );
}

// Animated heart like/unlike
export function LikeAnimation() {
  const [isLiked, setIsLiked] = useState(false);
  const [showBurst, setShowBurst] = useState(false);

  useEffect(() => {
    const interval = setInterval(() => {
      setIsLiked((prev) => {
        if (!prev) {
          setShowBurst(true);
          setTimeout(() => setShowBurst(false), 600);
        }
        return !prev;
      });
    }, 2000);
    return () => clearInterval(interval);
  }, []);

  return (
    <div className="h-full w-full flex items-center justify-center">
      <div className="relative">
        {/* Burst particles */}
        {showBurst && (
          <>
            {[...Array(6)].map((_, i) => (
              <Heart
                key={i}
                className="absolute h-4 w-4 text-red-500 fill-red-500 animate-ping"
                style={{
                  left: '50%',
                  top: '50%',
                  transform: `translate(-50%, -50%) rotate(${i * 60}deg) translateY(-30px)`,
                  animationDuration: '0.6s',
                  animationDelay: `${i * 0.05}s`,
                }}
              />
            ))}
          </>
        )}
        
        {/* Main heart */}
        <Heart
          className={cn(
            'h-16 w-16 transition-all duration-300',
            isLiked 
              ? 'text-red-500 fill-red-500 scale-110' 
              : 'text-muted-foreground scale-100'
          )}
          style={{
            filter: isLiked ? 'drop-shadow(0 0 8px rgba(239, 68, 68, 0.5))' : 'none',
          }}
        />
        
        {/* Ripple effect */}
        {showBurst && (
          <div 
            className="absolute inset-0 rounded-full border-2 border-red-500 animate-ping"
            style={{ 
              transform: 'scale(1.5)',
              animationDuration: '0.6s',
            }}
          />
        )}
      </div>
    </div>
  );
}

// Upload progress animation with success splash
export function UploadAnimation() {
  const [progress, setProgress] = useState(0);
  const [status, setStatus] = useState<'uploading' | 'done'>('uploading');
  const [showSplash, setShowSplash] = useState(false);

  useEffect(() => {
    const interval = setInterval(() => {
      setProgress((prev) => {
        if (prev >= 100) {
          setStatus('done');
          setShowSplash(true);
          setTimeout(() => {
            setShowSplash(false);
            setProgress(0);
            setStatus('uploading');
          }, 2000);
          return 100;
        }
        return prev + 5;
      });
    }, 100);
    return () => clearInterval(interval);
  }, []);

  return (
    <div className="h-full w-full p-3 flex items-center justify-center overflow-hidden">
      {/* Success splash background */}
      {showSplash && (
        <>
          {/* Radial glow */}
          <div 
            className="absolute inset-0 animate-in fade-in duration-300 pointer-events-none"
            style={{
              background: 'radial-gradient(circle at center, rgba(34, 197, 94, 0.25) 0%, transparent 70%)',
            }}
          />
          {/* Expanding rings */}
          <div className="absolute inset-0 flex items-center justify-center pointer-events-none">
            <div 
              className="absolute w-24 h-24 rounded-full border-2 border-green-500/40 animate-ping"
              style={{ animationDuration: '1s' }}
            />
            <div 
              className="absolute w-36 h-36 rounded-full border border-green-500/20 animate-ping"
              style={{ animationDuration: '1.5s', animationDelay: '0.2s' }}
            />
          </div>
          {/* Particle burst */}
          {[...Array(6)].map((_, i) => (
            <div
              key={i}
              className="absolute w-1.5 h-1.5 bg-green-500 rounded-full animate-ping pointer-events-none"
              style={{
                left: '50%',
                top: '50%',
                transform: `translate(-50%, -50%) rotate(${i * 60}deg) translateY(-45px)`,
                animationDuration: '0.8s',
                animationDelay: `${i * 0.05}s`,
              }}
            />
          ))}
        </>
      )}
      
      <Card className={cn(
        "w-full max-w-[140px] p-2.5 gap-0 transition-all duration-300",
        showSplash && "shadow-md shadow-green-500/20 border-green-500/50"
      )}>
        <div className={cn(
          "aspect-video rounded-md mb-2 flex items-center justify-center overflow-hidden transition-colors duration-300",
          showSplash ? "bg-green-500/10" : "bg-muted"
        )}>
          {status === 'done' ? (
            <div className="text-green-500 animate-in zoom-in duration-300">
              <svg className="h-6 w-6" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={2.5}>
                <path strokeLinecap="round" strokeLinejoin="round" d="M5 13l4 4L19 7" />
              </svg>
            </div>
          ) : (
            <div className="text-muted-foreground">
              <svg className="h-4 w-4 animate-bounce" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M7 16a4 4 0 01-.88-7.903A5 5 0 1115.9 6L16 6a5 5 0 011 9.9M15 13l-3-3m0 0l-3 3m3-3v12" />
              </svg>
            </div>
          )}
        </div>
        <div className="space-y-1">
          <div className="flex justify-between text-[8px]">
            <span className={cn(
              "transition-colors duration-300",
              showSplash ? "text-green-600 font-medium" : "text-muted-foreground"
            )}>
              {status === 'done' ? 'Selesai!' : 'Uploading...'}
            </span>
            <span className="tabular-nums">{progress}%</span>
          </div>
          <div className="h-1 bg-muted rounded-full overflow-hidden">
            <div
              className={cn(
                "h-full rounded-full transition-all duration-100",
                showSplash ? "bg-green-500" : "bg-primary"
              )}
              style={{ width: `${progress}%` }}
            />
          </div>
        </div>
      </Card>
    </div>
  );
}
