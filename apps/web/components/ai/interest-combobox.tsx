'use client';

import { useState, useRef, useEffect } from 'react';
import { Check, X, Plus } from 'lucide-react';
import { Badge } from '@/components/ui/badge';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import { cn } from '@/lib/utils';

interface InterestComboboxProps {
  selectedInterests: string[];
  onInterestsChange: (interests: string[]) => void;
  suggestions: string[];
  maxSelections?: number;
}

export function InterestCombobox({
  selectedInterests,
  onInterestsChange,
  suggestions,
  maxSelections = 10,
}: InterestComboboxProps) {
  const [inputValue, setInputValue] = useState('');
  const [isOpen, setIsOpen] = useState(false);
  const [filteredSuggestions, setFilteredSuggestions] = useState<string[]>([]);
  const inputRef = useRef<HTMLInputElement>(null);
  const dropdownRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    if (inputValue.trim()) {
      const filtered = suggestions.filter(
        (s) =>
          s.toLowerCase().includes(inputValue.toLowerCase()) &&
          !selectedInterests.includes(s)
      );
      setFilteredSuggestions(filtered);
      setIsOpen(filtered.length > 0);
    } else {
      setFilteredSuggestions([]);
      setIsOpen(false);
    }
  }, [inputValue, suggestions, selectedInterests]);

  useEffect(() => {
    const handleClickOutside = (event: MouseEvent) => {
      if (
        dropdownRef.current &&
        !dropdownRef.current.contains(event.target as Node) &&
        inputRef.current &&
        !inputRef.current.contains(event.target as Node)
      ) {
        setIsOpen(false);
      }
    };

    document.addEventListener('mousedown', handleClickOutside);
    return () => document.removeEventListener('mousedown', handleClickOutside);
  }, []);

  const handleAddInterest = (interest: string) => {
    if (
      interest.trim() &&
      !selectedInterests.includes(interest.trim()) &&
      selectedInterests.length < maxSelections
    ) {
      onInterestsChange([...selectedInterests, interest.trim()]);
      setInputValue('');
      setIsOpen(false);
      inputRef.current?.focus();
    }
  };

  const handleRemoveInterest = (interest: string) => {
    onInterestsChange(selectedInterests.filter((i) => i !== interest));
  };

  const handleKeyDown = (e: React.KeyboardEvent<HTMLInputElement>) => {
    if (e.key === 'Enter') {
      e.preventDefault();
      if (filteredSuggestions.length > 0) {
        handleAddInterest(filteredSuggestions[0]);
      } else if (inputValue.trim()) {
        handleAddInterest(inputValue);
      }
    } else if (e.key === 'Escape') {
      setIsOpen(false);
    }
  };

  const availableSuggestions = suggestions.filter((s) => !selectedInterests.includes(s));

  return (
    <div className="space-y-3">
      <Label htmlFor="interests" className="text-sm font-medium">
        Minat & Keahlian <span className="text-destructive">*</span>
        <span className="ml-2 text-xs font-normal text-muted-foreground">
          ({selectedInterests.length}/{maxSelections})
        </span>
      </Label>

      {/* Quick Add Suggestions */}
      {selectedInterests.length < maxSelections && availableSuggestions.length > 0 && (
        <div className="space-y-2">
          <p className="text-xs font-medium text-muted-foreground">Pilih cepat:</p>
          <div className="flex flex-wrap gap-2">
            {availableSuggestions.slice(0, 8).map((suggestion) => (
              <Badge
                key={suggestion}
                variant="outline"
                className="cursor-pointer text-xs transition-colors hover:border-primary hover:bg-primary/10 hover:text-primary"
                onClick={() => handleAddInterest(suggestion)}
              >
                <Plus className="mr-1 h-3 w-3" />
                {suggestion}
              </Badge>
            ))}
          </div>
        </div>
      )}

      {/* Input with Autocomplete */}
      <div className="relative">
        <div className="flex gap-2">
          <div className="relative flex-1">
            <Input
              ref={inputRef}
              id="interests"
              placeholder={
                selectedInterests.length >= maxSelections
                  ? 'Maksimal tercapai'
                  : 'Ketik atau pilih minat...'
              }
              value={inputValue}
              onChange={(e) => setInputValue(e.target.value)}
              onKeyDown={handleKeyDown}
              onFocus={() => {
                if (inputValue.trim() && filteredSuggestions.length > 0) {
                  setIsOpen(true);
                }
              }}
              disabled={selectedInterests.length >= maxSelections}
              className="h-10"
            />

            {/* Autocomplete Dropdown */}
            {isOpen && filteredSuggestions.length > 0 && (
              <div
                ref={dropdownRef}
                className="absolute top-full z-50 mt-1 max-h-60 w-full overflow-auto rounded-md border bg-popover p-1 shadow-lg"
              >
                {filteredSuggestions.map((suggestion) => (
                  <button
                    key={suggestion}
                    type="button"
                    onClick={() => handleAddInterest(suggestion)}
                    className="flex w-full items-center gap-2 rounded-sm px-3 py-2 text-sm transition-colors hover:bg-accent hover:text-accent-foreground"
                  >
                    <Plus className="h-4 w-4 text-muted-foreground" />
                    {suggestion}
                  </button>
                ))}
              </div>
            )}
          </div>

          <Button
            type="button"
            onClick={() => inputValue.trim() && handleAddInterest(inputValue)}
            variant="outline"
            size="icon"
            className="h-10 w-10 shrink-0"
            disabled={!inputValue.trim() || selectedInterests.length >= maxSelections}
          >
            <Plus className="h-4 w-4" />
          </Button>
        </div>
      </div>

      {/* Selected Interests */}
      {selectedInterests.length > 0 && (
        <div className="flex flex-wrap gap-2 rounded-md border bg-muted/30 p-3">
          {selectedInterests.map((interest) => (
            <Badge
              key={interest}
              variant="secondary"
              className="gap-1.5 pl-2.5 pr-1.5 transition-colors hover:bg-secondary/80"
            >
              {interest}
              <button
                type="button"
                onClick={() => handleRemoveInterest(interest)}
                className="ml-0.5 rounded-sm opacity-70 transition-opacity hover:opacity-100"
              >
                <X className="h-3 w-3" />
              </button>
            </Badge>
          ))}
        </div>
      )}
    </div>
  );
}
