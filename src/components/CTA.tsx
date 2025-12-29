import { Button } from "@/components/ui/button";
import { ArrowRight } from "lucide-react";

const CTA = () => {
  return (
    <section className="py-24 relative overflow-hidden">
      {/* Background Effects */}
      <div className="absolute inset-0 bg-gradient-to-br from-primary/5 via-transparent to-primary/10" />
      <div className="absolute top-1/2 left-1/2 -translate-x-1/2 -translate-y-1/2 w-[600px] h-[600px] bg-primary/10 rounded-full blur-[150px]" />

      <div className="container mx-auto px-6 relative z-10">
        <div className="max-w-4xl mx-auto text-center">
          <h2 className="text-3xl sm:text-4xl lg:text-5xl font-bold text-foreground mb-6">
            Ready to Maximize Your{" "}
            <span className="text-primary">Digital Asset Returns?</span>
          </h2>
          <p className="text-lg text-muted-foreground max-w-2xl mx-auto mb-10">
            Join sophisticated investors who trust Monera Digital for institutional-grade 
            digital asset management. Start earning premium yields today.
          </p>

          <div className="flex flex-col sm:flex-row items-center justify-center gap-4">
            <Button variant="hero" size="xl">
              Schedule Consultation
              <ArrowRight className="ml-2" />
            </Button>
            <Button variant="heroOutline" size="xl">
              View Documentation
            </Button>
          </div>

          <p className="text-sm text-muted-foreground mt-8">
            Minimum deposit: $100,000 • Dedicated relationship manager • 24/7 support
          </p>
        </div>
      </div>
    </section>
  );
};

export default CTA;
