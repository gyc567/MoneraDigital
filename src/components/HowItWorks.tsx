import { UserCheck, Send, TrendingUp, Wallet } from "lucide-react";

const steps = [
  {
    number: "01",
    icon: UserCheck,
    title: "Onboarding & KYC",
    description: "Complete our streamlined verification process designed for high-net-worth individuals with institutional-grade compliance.",
  },
  {
    number: "02",
    icon: Send,
    title: "Deposit Assets",
    description: "Transfer your digital assets to our secure cold storage infrastructure with full segregation and real-time monitoring.",
  },
  {
    number: "03",
    icon: TrendingUp,
    title: "Earn Yields",
    description: "Your assets generate returns through our sophisticated market strategies while you maintain full transparency and control.",
  },
  {
    number: "04",
    icon: Wallet,
    title: "Withdraw Anytime",
    description: "Access your principal and earned yields according to your chosen terms with complete flexibility and no hidden fees.",
  },
];

const HowItWorks = () => {
  return (
    <section id="solutions" className="py-24 relative">
      <div className="container mx-auto px-6">
        {/* Section Header */}
        <div className="text-center max-w-2xl mx-auto mb-16">
          <span className="text-primary text-sm font-medium tracking-wider uppercase mb-4 block">
            Simple Process
          </span>
          <h2 className="text-3xl sm:text-4xl lg:text-5xl font-bold text-foreground mb-6">
            How It Works
          </h2>
          <p className="text-muted-foreground text-lg">
            A seamless journey from onboarding to earning institutional-grade yields 
            on your digital assets.
          </p>
        </div>

        {/* Steps */}
        <div className="max-w-4xl mx-auto">
          <div className="relative">
            {/* Connecting Line */}
            <div className="absolute left-8 top-0 bottom-0 w-px bg-gradient-to-b from-primary/50 via-primary/20 to-transparent hidden md:block" />

            <div className="space-y-12">
              {steps.map((step, index) => (
                <div
                  key={step.number}
                  className="relative flex gap-6 md:gap-8"
                  style={{ animationDelay: `${index * 0.15}s` }}
                >
                  {/* Step Number & Icon */}
                  <div className="flex-shrink-0 relative z-10">
                    <div className="w-16 h-16 rounded-2xl bg-card border border-border flex items-center justify-center group hover:border-primary/50 transition-colors">
                      <step.icon className="w-7 h-7 text-primary" />
                    </div>
                  </div>

                  {/* Content */}
                  <div className="flex-1 pt-2">
                    <div className="flex items-baseline gap-3 mb-3">
                      <span className="text-primary font-mono text-sm">{step.number}</span>
                      <h3 className="text-xl font-semibold text-foreground">
                        {step.title}
                      </h3>
                    </div>
                    <p className="text-muted-foreground leading-relaxed max-w-lg">
                      {step.description}
                    </p>
                  </div>
                </div>
              ))}
            </div>
          </div>
        </div>
      </div>
    </section>
  );
};

export default HowItWorks;
