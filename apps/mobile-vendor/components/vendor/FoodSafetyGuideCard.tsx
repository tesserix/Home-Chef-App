import { useState } from "react";
import { Linking, Pressable, StyleSheet, Text, View } from "react-native";
import { theme } from "@homechef/mobile-shared/theme";
import { useToast } from "@homechef/mobile-shared/ui";
import type { MarketCode } from "../../lib/market";

const guides = {
  AU: {
    country: "Australia",
    steps: [
      [
        "Contact your local council or food regulator",
        "Tell them your kitchen address, menu and how you will sell food. Ask which food-business registration, licence or notification applies to your home kitchen. ABLIS can help you find local requirements.",
      ],
      [
        "Prepare your kitchen and apply",
        "Follow the regulator’s food safety and premises requirements. Ask about inspections, food-handler training and whether you need a food safety supervisor. Fees and requirements vary by state, council and activity.",
      ],
      [
        "Upload your registration",
        "Upload a clear photo or PDF of the current food-business registration, licence or official confirmation issued by your council or food authority. Include every page showing the business, kitchen address and registration details.",
      ],
    ],
    links: [
      ["Find local requirements with ABLIS", "https://ablis.business.gov.au/"],
      [
        "FSANZ home-based food business guide",
        "https://www.foodstandards.gov.au/business/food-safety/home-based-industry",
      ],
    ],
  },
  NZ: {
    country: "New Zealand",
    steps: [
      [
        "Check MPI My Food Rules",
        "Use My Food Rules to find out which Food Control Plan or National Programme applies to your menu and food activities, and who you must register with.",
      ],
      [
        "Register with your council or MPI",
        "Most food-service businesses operating from one council area register with that council. Custom Food Control Plans register with MPI. Follow the tool’s result, complete the application and arrange verification; some programmes require a verifier’s letter before registration.",
      ],
      [
        "Upload your registration",
        "Upload a clear photo or PDF of your current food-business registration certificate from your council or MPI. Include every page showing the business, kitchen address and registration details. Keep your food safety records and complete the required verification.",
      ],
    ],
    links: [
      [
        "MPI My Food Rules",
        "https://www.mpi.govt.nz/food-business/food-safety-rules",
      ],
      [
        "MPI registration and verification guide",
        "https://www.mpi.govt.nz/food-business/starting-a-food-business/register-food-business",
      ],
      [
        "Find your local NZ council",
        "https://www.lgnz.co.nz/local-government-in-nz/new-zealands-councils/",
      ],
    ],
  },
};

export function FoodSafetyGuideCard({ country }: { country: MarketCode }) {
  const [expanded, setExpanded] = useState(false);
  const { show } = useToast();
  if (country === "IN") return null;
  const guide = guides[country];

  async function openLink(url: string) {
    try {
      await Linking.openURL(url);
    } catch {
      show({
        message: "Could not open the official guide. Please try again.",
        tone: "error",
      });
    }
  }

  return (
    <View style={styles.card}>
      <Pressable
        accessibilityRole="button"
        accessibilityLabel="Food safety registration guide"
        accessibilityState={{ expanded }}
        onPress={() => setExpanded(!expanded)}
        style={styles.toggle}
      >
        <View style={styles.heading}>
          <Text style={styles.title}>Need a food safety certificate?</Text>
          <Text style={styles.body}>
            {guide.country} · Council registration and upload guide
          </Text>
        </View>
        <Text style={styles.title}>{expanded ? "−" : "+"}</Text>
      </Pressable>
      {expanded ? (
        <View style={styles.details}>
          {guide.steps.map(([title, body], index) => (
            <View key={title} style={styles.step}>
              <Text style={styles.title}>
                {index + 1}. {title}
              </Text>
              <Text style={styles.body}>{body}</Text>
            </View>
          ))}
          <Text style={styles.body}>
            A food-handler or supervisor training certificate does not replace
            food-business registration. Enter the registration number and expiry
            date shown on your document, where applicable. If your authority
            confirms an exemption or issues a different document, contact Fe3dr
            support for review.
          </Text>
          <Text style={styles.body}>
            Still applying? You can save and continue onboarding, then return to
            Documents to upload it. Fe3dr approval does not replace council or
            regulator approval to sell food.
          </Text>
          {guide.links.map(([label, url]) => (
            <Pressable
              key={url}
              accessibilityRole="link"
              accessibilityLabel={`Open ${label}`}
              onPress={() => openLink(url)}
              style={styles.link}
            >
              <Text style={styles.linkText}>{label} ↗</Text>
            </Pressable>
          ))}
        </View>
      ) : null}
    </View>
  );
}

const styles = StyleSheet.create({
  card: {
    backgroundColor: theme.colors.paper,
    borderRadius: theme.radius.lg,
    padding: theme.spacing[4],
    ...theme.shadow[1],
  },
  toggle: {
    minHeight: 44,
    flexDirection: "row",
    alignItems: "center",
    gap: theme.spacing[3],
  },
  heading: { flex: 1, gap: theme.spacing[1] },
  title: {
    fontFamily: "Inter-SemiBold",
    fontSize: 15,
    lineHeight: 21,
    color: theme.colors.ink.DEFAULT,
  },
  body: {
    fontFamily: "Inter",
    fontSize: 14,
    lineHeight: 21,
    color: theme.colors.ink.soft,
  },
  details: { gap: theme.spacing[4], paddingTop: theme.spacing[4] },
  step: { gap: theme.spacing[1] },
  link: { minHeight: 44, justifyContent: "center" },
  linkText: {
    fontFamily: "Inter-SemiBold",
    fontSize: 14,
    lineHeight: 21,
    color: theme.colors.ink.DEFAULT,
    textDecorationLine: "underline",
  },
});
