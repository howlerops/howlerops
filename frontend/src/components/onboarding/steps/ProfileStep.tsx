import { User } from "lucide-react";
import { useState } from "react";

import { Button } from "@/components/ui/button";
import { Checkbox } from "@/components/ui/checkbox";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import {
  USE_CASE_LABELS,
  UseCase,
  USER_ROLE_LABELS,
  UserProfile,
  UserRole,
} from "@/types/onboarding";

interface ProfileStepProps {
  onNext: (profile: UserProfile) => void;
  onBack: () => void;
  initialProfile?: UserProfile;
}

export function ProfileStep({
  onNext,
  onBack,
  initialProfile,
}: ProfileStepProps) {
  const [name, setName] = useState(initialProfile?.name || "");
  const [useCases, setUseCases] = useState<UseCase[]>(
    initialProfile?.useCases || []
  );
  const [role, setRole] = useState<UserRole | "">(initialProfile?.role || "");

  const handleUseCaseToggle = (useCase: UseCase) => {
    setUseCases((prev) =>
      prev.includes(useCase)
        ? prev.filter((uc) => uc !== useCase)
        : [...prev, useCase]
    );
  };

  const handleSubmit = () => {
    if (role) {
      onNext({
        name: name || undefined,
        useCases,
        role: role as UserRole,
      });
    }
  };

  const canProceed = role && useCases.length > 0;

  return (
    <div className="max-w-lg mx-auto space-y-6 py-8">
      <div className="text-center space-y-2 mb-8">
        <div className="w-16 h-16 rounded-full bg-primary/10 flex items-center justify-center mx-auto mb-4">
          <User className="w-8 h-8 text-primary" />
        </div>
        <h2 className="text-2xl font-bold">Tell us about yourself</h2>
        <p className="text-muted-foreground">
          Help us personalize your experience
        </p>
      </div>

      <div className="space-y-4">
        <div className="space-y-2">
          <Label htmlFor="name">Name (optional)</Label>
          <Input
            id="name"
            placeholder="Enter your name"
            value={name}
            onChange={(e) => setName(e.target.value)}
          />
        </div>

        <div className="space-y-2">
          <Label>What will you use Howlerops for?</Label>
          <div className="space-y-2">
            {Object.entries(USE_CASE_LABELS).map(([value, label]) => (
              <div key={value} className="flex items-center space-x-2">
                <Checkbox
                  id={value}
                  checked={useCases.includes(value as UseCase)}
                  onCheckedChange={() => handleUseCaseToggle(value as UseCase)}
                />
                <Label
                  htmlFor={value}
                  className="text-sm font-normal cursor-pointer"
                >
                  {label}
                </Label>
              </div>
            ))}
          </div>
        </div>

        <div className="space-y-2">
          <Label htmlFor="role">What's your role?</Label>
          <Select
            value={role}
            onValueChange={(value) => setRole(value as UserRole)}
          >
            <SelectTrigger id="role">
              <SelectValue placeholder="Select your role" />
            </SelectTrigger>
            <SelectContent>
              {Object.entries(USER_ROLE_LABELS).map(([value, label]) => (
                <SelectItem key={value} value={value}>
                  {label}
                </SelectItem>
              ))}
            </SelectContent>
          </Select>
        </div>
      </div>

      <div className="flex items-center gap-3 pt-4">
        <Button variant="outline" onClick={onBack}>
          Back
        </Button>
        <Button
          onClick={handleSubmit}
          disabled={!canProceed}
          className="flex-1"
        >
          Continue
        </Button>
      </div>
    </div>
  );
}
