"""
Nim Onboarding Animation
========================
Explains the Stabilize → Save → Invest flow using Manim.

Run with:
    manim -pql nim_onboarding.py NimOnboarding

For high quality:
    manim -pqh nim_onboarding.py NimOnboarding
"""

from manim import *

# Color palette
NIM_BLUE = "#007AFF"
NIM_GREEN = "#34C759"
NIM_RED = "#FF3B30"
NIM_ORANGE = "#FF9500"
NIM_PURPLE = "#AF52DE"
NIM_TEAL = "#5AC8FA"
NIM_GRAY = "#8E8E93"
NIM_DARK = "#1C1C1E"
NIM_LIGHT = "#F2F2F7"

# Typography settings
FONT_TITLE = 56
FONT_HEADING = 40
FONT_SUBHEADING = 28
FONT_BODY = 22
FONT_SMALL = 18
FONT_CAPTION = 14

# Spacing
SPACING_XL = 1.0
SPACING_LG = 0.6
SPACING_MD = 0.4
SPACING_SM = 0.25
SPACING_XS = 0.15


class NimOnboarding(Scene):
    """Main onboarding animation explaining the Nim flow."""

    def construct(self):
        self.camera.background_color = NIM_DARK

        # Scene 1: Title
        self.play_title()

        # Scene 2: The Problem
        self.play_problem()

        # Scene 3: The Solution
        self.play_solution()

        # Scene 4: The Journey Overview
        self.play_journey_overview()

        # Scene 5-10: Individual Steps
        self.play_step_chat()
        self.play_step_budget()
        self.play_step_subscriptions()
        self.play_step_savings()
        self.play_step_trading()
        self.play_step_autopilot()

        # Scene 11: The Flywheel
        self.play_flywheel()

        # Scene 12: Call to Action
        self.play_cta()

    def play_title(self):
        """Animated title sequence."""
        logo = Text("Nim", font_size=96, weight=BOLD, color=WHITE)
        logo.set_color_by_gradient(NIM_BLUE, NIM_PURPLE)

        tagline = Text(
            "Money, Multiplied", font_size=FONT_SUBHEADING, color=NIM_GRAY, slant=ITALIC
        )
        tagline.next_to(logo, DOWN, buff=SPACING_MD)

        group = VGroup(logo, tagline)

        self.play(Write(logo, run_time=1.5))
        self.play(FadeIn(tagline, shift=UP * 0.3))
        self.wait(1.5)
        self.play(FadeOut(group, shift=UP))

    def play_problem(self):
        """Show the problem with traditional finance apps."""
        title = Text("The Problem", font_size=FONT_HEADING, color=NIM_RED, weight=BOLD)
        title.to_edge(UP, buff=SPACING_LG)

        self.play(Write(title))

        # Traditional flow (wrong way)
        steps = ["Trading", "Budget", "Savings"]
        colors = [NIM_RED, NIM_GRAY, NIM_GRAY]

        boxes = []
        for step, color in zip(steps, colors):
            box = RoundedRectangle(
                width=2.8,
                height=1.4,
                corner_radius=0.2,
                fill_color=color,
                fill_opacity=0.2,
                stroke_color=color,
                stroke_width=3,
            )
            text = Text(step, font_size=FONT_BODY, color=WHITE, weight=BOLD)
            text.move_to(box)
            group = VGroup(box, text)
            boxes.append(group)

        wrong_flow = VGroup(*boxes).arrange(RIGHT, buff=SPACING_LG)
        wrong_flow.move_to(ORIGIN + UP * 0.3)

        # Arrows between boxes
        arrows = VGroup()
        for i in range(len(boxes) - 1):
            arrow = Arrow(
                boxes[i].get_right(),
                boxes[i + 1].get_left(),
                buff=0.15,
                color=NIM_GRAY,
                stroke_width=3,
            )
            arrows.add(arrow)

        self.play(LaggedStart(*[FadeIn(b, scale=0.8) for b in boxes], lag_ratio=0.2))
        self.play(Create(arrows))

        # Anxiety indicator
        anxiety_group = VGroup()
        anxiety = Text("This creates anxiety", font_size=FONT_BODY, color=NIM_RED)
        worry = Text("😰", font_size=36)
        anxiety_group = VGroup(anxiety, worry).arrange(RIGHT, buff=SPACING_SM)
        anxiety_group.next_to(wrong_flow, DOWN, buff=SPACING_XL)

        self.play(FadeIn(anxiety_group, shift=UP * 0.3))

        # X mark over the flow
        x_mark = Cross(wrong_flow, stroke_color=NIM_RED, stroke_width=6)
        self.play(Create(x_mark))

        self.wait(1)
        self.play(FadeOut(Group(*self.mobjects)))

    def play_solution(self):
        """Show Nim's solution."""
        title = Text(
            "The Nim Way", font_size=FONT_HEADING, color=NIM_GREEN, weight=BOLD
        )
        title.to_edge(UP, buff=SPACING_LG)

        self.play(Write(title))

        # Three pillars
        pillar_data = [
            ("1", "Stabilize", "Budget and cut waste", NIM_BLUE),
            ("2", "Save", "Automate transfers", NIM_GREEN),
            ("3", "Invest", "Only the surplus", NIM_PURPLE),
        ]

        pillars = []
        for num, label, desc, color in pillar_data:
            # Pillar container
            box = RoundedRectangle(
                width=3.2,
                height=2.4,
                corner_radius=0.2,
                fill_color=color,
                fill_opacity=0.15,
                stroke_color=color,
                stroke_width=3,
            )

            # Number badge
            num_circle = Circle(
                radius=0.3, fill_color=color, fill_opacity=1, stroke_width=0
            )
            num_text = Text(num, font_size=FONT_BODY, color=WHITE, weight=BOLD)
            num_text.move_to(num_circle)
            num_group = VGroup(num_circle, num_text)
            num_group.move_to(box.get_top() + DOWN * 0.5)

            # Label
            label_text = Text(
                label, font_size=FONT_SUBHEADING, color=WHITE, weight=BOLD
            )
            label_text.next_to(num_group, DOWN, buff=SPACING_SM)

            # Description
            desc_text = Text(desc, font_size=FONT_SMALL, color=NIM_GRAY)
            desc_text.next_to(label_text, DOWN, buff=SPACING_XS)

            pillar = VGroup(box, num_group, label_text, desc_text)
            pillars.append(pillar)

        pillars_group = VGroup(*pillars).arrange(RIGHT, buff=SPACING_MD)
        pillars_group.move_to(ORIGIN)

        # Animate pillars rising
        for pillar in pillars:
            pillar.shift(DOWN * 2)

        self.play(
            LaggedStart(
                *[pillar.animate.shift(UP * 2) for pillar in pillars], lag_ratio=0.25
            )
        )

        # Arrows between pillars
        for i in range(len(pillars) - 1):
            arrow = Arrow(
                pillars[i].get_right() + LEFT * 0.3,
                pillars[i + 1].get_left() + RIGHT * 0.3,
                buff=0.1,
                color=WHITE,
                stroke_width=3,
            )
            self.play(GrowArrow(arrow), run_time=0.4)

        # Confidence message
        confidence_group = VGroup()
        confidence = Text(
            "This builds confidence", font_size=FONT_BODY, color=NIM_GREEN
        )
        happy = Text("😊", font_size=36)
        confidence_group = VGroup(confidence, happy).arrange(RIGHT, buff=SPACING_SM)
        confidence_group.next_to(pillars_group, DOWN, buff=SPACING_XL)

        self.play(FadeIn(confidence_group, shift=UP * 0.3))

        self.wait(1.5)
        self.play(FadeOut(Group(*self.mobjects)))

    def play_journey_overview(self):
        """Show the complete journey."""
        title = Text("Your Journey", font_size=FONT_HEADING, color=WHITE, weight=BOLD)
        title.to_edge(UP, buff=SPACING_LG)

        self.play(Write(title))

        # Journey steps in a circle
        steps_data = [
            ("Chat", NIM_BLUE),
            ("Budget", NIM_TEAL),
            ("Subscriptions", NIM_ORANGE),
            ("Savings", NIM_GREEN),
            ("Investing", NIM_PURPLE),
            ("Autopilot", NIM_GRAY),
        ]

        steps = []
        n = len(steps_data)
        radius = 2.4

        for i, (label, color) in enumerate(steps_data):
            angle = PI / 2 - (2 * PI * i / n)
            pos = radius * np.array([np.cos(angle), np.sin(angle), 0])

            # Circle node
            circle = Circle(
                radius=0.55,
                fill_color=color,
                fill_opacity=0.3,
                stroke_color=color,
                stroke_width=3,
            )
            circle.move_to(pos)

            # Label
            label_text = Text(label, font_size=FONT_CAPTION, color=WHITE, weight=BOLD)
            label_text.next_to(circle, DOWN, buff=SPACING_XS)

            step = VGroup(circle, label_text)
            steps.append(step)

        # Center Nim logo
        center_circle = Circle(
            radius=0.9,
            fill_color=NIM_BLUE,
            fill_opacity=0.4,
            stroke_color=NIM_BLUE,
            stroke_width=3,
        )
        center_text = Text("Nim", font_size=FONT_SUBHEADING, color=WHITE, weight=BOLD)
        center_text.move_to(center_circle)
        center = VGroup(center_circle, center_text)

        self.play(FadeIn(center, scale=0.5))
        self.play(LaggedStart(*[FadeIn(s, scale=0.5) for s in steps], lag_ratio=0.12))

        # Connecting arrows
        arrows = []
        for i in range(n):
            start = steps[i][0].get_center()
            end = steps[(i + 1) % n][0].get_center()

            arrow = CurvedArrow(start, end, color=WHITE, stroke_width=2, angle=TAU / 10)
            arrows.append(arrow)

        self.play(LaggedStart(*[Create(a) for a in arrows], lag_ratio=0.08))

        self.wait(1.5)
        self.play(FadeOut(Group(*self.mobjects)))

    def play_step_chat(self):
        """Step 1: Chat with Nim."""
        self.play_step_intro("1", "Chat with Nim", "Turn intent into a plan", NIM_BLUE)

        # Chat mockup
        chat_bg = RoundedRectangle(
            width=5.5,
            height=4.5,
            corner_radius=0.3,
            fill_color="#2C2C2E",
            fill_opacity=1,
            stroke_color=NIM_BLUE,
            stroke_width=2,
        )
        chat_bg.shift(LEFT * 1.5)

        # User message
        user_bubble = RoundedRectangle(
            width=4,
            height=0.9,
            corner_radius=0.3,
            fill_color=NIM_BLUE,
            fill_opacity=0.8,
            stroke_width=0,
        )
        user_text = Text(
            "I want to save £500 per month", font_size=FONT_CAPTION, color=WHITE
        )
        user_text.move_to(user_bubble)
        user_msg = VGroup(user_bubble, user_text)
        user_msg.move_to(chat_bg.get_center() + UP * 1.2 + RIGHT * 0.3)

        # Nim response
        nim_bubble = RoundedRectangle(
            width=4.5,
            height=1.6,
            corner_radius=0.3,
            fill_color="#3A3A3C",
            fill_opacity=1,
            stroke_width=0,
        )
        nim_lines = VGroup(
            Text(
                "Great goal! Quick questions:",
                font_size=FONT_CAPTION,
                color=WHITE,
                weight=BOLD,
            ),
            Text("1. When do you get paid?", font_size=FONT_CAPTION, color=NIM_GRAY),
            Text(
                "2. What's your risk comfort?", font_size=FONT_CAPTION, color=NIM_GRAY
            ),
        ).arrange(DOWN, aligned_edge=LEFT, buff=SPACING_XS)
        nim_lines.move_to(nim_bubble)
        nim_msg = VGroup(nim_bubble, nim_lines)
        nim_msg.move_to(chat_bg.get_center() + DOWN * 0.6 + LEFT * 0.2)

        chat = VGroup(chat_bg, user_msg, nim_msg)
        chat.scale(0.85)

        self.play(FadeIn(chat_bg, scale=0.95))
        self.play(FadeIn(user_msg, shift=LEFT * 0.3))
        self.wait(0.3)
        self.play(FadeIn(nim_msg, shift=RIGHT * 0.3))

        # Output panel
        output_title = Text("Output", font_size=FONT_BODY, color=NIM_GREEN, weight=BOLD)
        output_title.next_to(chat_bg, RIGHT, buff=SPACING_LG)
        output_title.align_to(chat_bg, UP)

        outputs = VGroup(
            Text("✓  Goals defined", font_size=FONT_SMALL, color=WHITE),
            Text("✓  Rules created", font_size=FONT_SMALL, color=WHITE),
            Text("✓  Accounts linked", font_size=FONT_SMALL, color=WHITE),
        ).arrange(DOWN, aligned_edge=LEFT, buff=SPACING_SM)
        outputs.next_to(output_title, DOWN, buff=SPACING_MD, aligned_edge=LEFT)

        self.play(Write(output_title))
        self.play(
            LaggedStart(
                *[FadeIn(o, shift=RIGHT * 0.2) for o in outputs], lag_ratio=0.15
            )
        )

        self.wait(1)
        self.play(FadeOut(Group(*self.mobjects)))

    def play_step_budget(self):
        """Step 2: Budget Planner."""
        self.play_step_intro("2", "Budget Planner", "Make the plan real", NIM_TEAL)

        # Budget categories
        categories = [
            ("Needs", 0.45, NIM_BLUE, "£1,080"),
            ("Wants", 0.25, NIM_PURPLE, "£600"),
            ("Bills", 0.20, NIM_ORANGE, "£480"),
            ("Goals", 0.10, NIM_GREEN, "£240"),
        ]

        bars = VGroup()
        max_width = 6

        for name, pct, color, amount in categories:
            # Category label
            label = Text(name, font_size=FONT_SMALL, color=WHITE, weight=BOLD)

            # Progress bar background
            bar_bg = RoundedRectangle(
                width=max_width,
                height=0.5,
                corner_radius=0.1,
                fill_color="#3A3A3C",
                fill_opacity=1,
                stroke_width=0,
            )

            # Progress bar fill
            bar_fill = RoundedRectangle(
                width=max_width * pct,
                height=0.5,
                corner_radius=0.1,
                fill_color=color,
                fill_opacity=0.8,
                stroke_width=0,
            )
            bar_fill.align_to(bar_bg, LEFT)

            # Percentage
            pct_text = Text(
                f"{int(pct * 100)}%", font_size=FONT_CAPTION, color=color, weight=BOLD
            )

            # Amount
            amount_text = Text(amount, font_size=FONT_CAPTION, color=NIM_GRAY)

            # Arrange row
            label.next_to(bar_bg, LEFT, buff=SPACING_SM)
            pct_text.move_to(bar_bg.get_center())
            amount_text.next_to(bar_bg, RIGHT, buff=SPACING_SM)

            row = VGroup(label, bar_bg, bar_fill, pct_text, amount_text)
            bars.add(row)

        bars.arrange(DOWN, buff=SPACING_MD, aligned_edge=LEFT)
        bars.move_to(ORIGIN + LEFT * 0.5)

        # Animate bars
        for bar in bars:
            label, bar_bg, bar_fill, pct_text, amount_text = bar
            self.play(FadeIn(label), FadeIn(bar_bg), run_time=0.2)
            self.play(
                GrowFromEdge(bar_fill, LEFT),
                FadeIn(pct_text),
                FadeIn(amount_text),
                run_time=0.3,
            )

        # Guardrails section
        guardrails_box = RoundedRectangle(
            width=3.5,
            height=2.5,
            corner_radius=0.2,
            fill_color=NIM_ORANGE,
            fill_opacity=0.1,
            stroke_color=NIM_ORANGE,
            stroke_width=2,
        )
        guardrails_box.next_to(bars, RIGHT, buff=SPACING_LG)

        guardrails_title = Text(
            "Guardrails", font_size=FONT_BODY, color=NIM_ORANGE, weight=BOLD
        )
        guardrails_title.move_to(guardrails_box.get_top() + DOWN * 0.4)

        guardrails = VGroup(
            Text("Soft alerts at 80%", font_size=FONT_CAPTION, color=WHITE),
            Text("Hard stops optional", font_size=FONT_CAPTION, color=WHITE),
            Text("Real-time tracking", font_size=FONT_CAPTION, color=WHITE),
        ).arrange(DOWN, aligned_edge=LEFT, buff=SPACING_XS)
        guardrails.next_to(guardrails_title, DOWN, buff=SPACING_SM)

        self.play(FadeIn(guardrails_box), Write(guardrails_title))
        self.play(LaggedStart(*[FadeIn(g) for g in guardrails], lag_ratio=0.1))

        self.wait(1)
        self.play(FadeOut(Group(*self.mobjects)))

    def play_step_subscriptions(self):
        """Step 3: Subscriptions."""
        self.play_step_intro("3", "Subscriptions", "Kill silent drains", NIM_ORANGE)

        # Subscription cards
        subs_data = [
            ("Netflix", "£15.99/mo", "Used 3 days ago", "Keep", NIM_GREEN),
            ("Headspace", "£12.99/mo", "Not used in 67 days", "Cancel?", NIM_RED),
            ("Spotify", "£10.99/mo", "Price increased 10%", "Review", NIM_ORANGE),
        ]

        cards = VGroup()
        for name, price, usage, action, color in subs_data:
            card = RoundedRectangle(
                width=5,
                height=1.3,
                corner_radius=0.15,
                fill_color="#2C2C2E",
                fill_opacity=1,
                stroke_color=color,
                stroke_width=2,
            )

            # Service name
            name_text = Text(name, font_size=FONT_BODY, color=WHITE, weight=BOLD)
            name_text.move_to(card.get_left() + RIGHT * 1)

            # Price
            price_text = Text(price, font_size=FONT_SMALL, color=color)
            price_text.next_to(name_text, RIGHT, buff=SPACING_LG)

            # Usage info
            usage_text = Text(usage, font_size=FONT_CAPTION, color=NIM_GRAY)
            usage_text.next_to(name_text, DOWN, buff=SPACING_XS, aligned_edge=LEFT)

            # Action badge
            action_text = Text(action, font_size=FONT_CAPTION, color=color, weight=BOLD)
            action_text.move_to(card.get_right() + LEFT * 0.8)

            card_group = VGroup(card, name_text, price_text, usage_text, action_text)
            cards.add(card_group)

        cards.arrange(DOWN, buff=SPACING_SM)
        cards.move_to(ORIGIN + LEFT * 1.5)

        self.play(
            LaggedStart(*[FadeIn(c, shift=RIGHT * 0.3) for c in cards], lag_ratio=0.2)
        )

        # Savings potential box
        savings_box = RoundedRectangle(
            width=3.5,
            height=2.8,
            corner_radius=0.2,
            fill_color=NIM_GREEN,
            fill_opacity=0.15,
            stroke_color=NIM_GREEN,
            stroke_width=2,
        )
        savings_box.next_to(cards, RIGHT, buff=SPACING_LG)

        savings_content = VGroup(
            Text("Potential Savings", font_size=FONT_SMALL, color=NIM_GREEN),
            Text("£156", font_size=FONT_TITLE, color=WHITE, weight=BOLD),
            Text("per year", font_size=FONT_SMALL, color=NIM_GRAY),
        ).arrange(DOWN, buff=SPACING_XS)
        savings_content.move_to(savings_box)

        savings = VGroup(savings_box, savings_content)

        self.play(FadeIn(savings, scale=0.9))

        self.wait(1)
        self.play(FadeOut(Group(*self.mobjects)))

    def play_step_savings(self):
        """Step 4: Smart Savings."""
        self.play_step_intro("4", "Smart Savings", "Automate the wins", NIM_GREEN)

        # Left side: Rules
        rules_title = Text("Your Rules", font_size=FONT_BODY, color=WHITE, weight=BOLD)
        rules_title.to_edge(LEFT, buff=1.2).shift(UP * 1.2)

        rules_data = [
            ("Payday sweep", "Move £500 to savings"),
            ("Round-ups", "Save spare change"),
            ("Under budget", "Sweep 30% of surplus"),
        ]

        rules = VGroup()
        for title, desc in rules_data:
            rule_title = Text(title, font_size=FONT_SMALL, color=NIM_GREEN, weight=BOLD)
            rule_desc = Text(desc, font_size=FONT_CAPTION, color=NIM_GRAY)
            rule = VGroup(rule_title, rule_desc).arrange(
                DOWN, aligned_edge=LEFT, buff=SPACING_XS
            )
            rules.add(rule)

        rules.arrange(DOWN, aligned_edge=LEFT, buff=SPACING_MD)
        rules.next_to(rules_title, DOWN, buff=SPACING_MD, aligned_edge=LEFT)

        self.play(Write(rules_title))
        self.play(
            LaggedStart(*[FadeIn(r, shift=RIGHT * 0.2) for r in rules], lag_ratio=0.15)
        )

        # Right side: Emergency Fund Ladder
        ladder_title = Text(
            "Emergency Fund", font_size=FONT_BODY, color=NIM_GREEN, weight=BOLD
        )
        ladder_title.to_edge(RIGHT, buff=1.5).shift(UP * 1.2)

        stages = [
            ("Stage 1", "2 weeks", 1.0),
            ("Stage 2", "1 month", 0.6),
            ("Stage 3", "3 months", 0.0),
        ]

        ladder = VGroup()
        for stage, duration, progress in stages:
            # Background bar
            bg = RoundedRectangle(
                width=4,
                height=0.5,
                corner_radius=0.1,
                fill_color="#3A3A3C",
                fill_opacity=1,
                stroke_width=0,
            )

            # Progress fill
            if progress > 0:
                fill = RoundedRectangle(
                    width=4 * progress,
                    height=0.5,
                    corner_radius=0.1,
                    fill_color=NIM_GREEN,
                    fill_opacity=0.8,
                    stroke_width=0,
                )
                fill.align_to(bg, LEFT)
            else:
                fill = VGroup()

            # Labels
            stage_label = Text(
                f"{stage}: {duration}", font_size=FONT_CAPTION, color=WHITE
            )
            stage_label.next_to(bg, UP, buff=SPACING_XS, aligned_edge=LEFT)

            # Progress indicator
            if progress == 1.0:
                prog_text = Text(
                    "Complete", font_size=FONT_CAPTION, color=NIM_GREEN, weight=BOLD
                )
            elif progress > 0:
                prog_text = Text(
                    f"{int(progress * 100)}%", font_size=FONT_CAPTION, color=NIM_GREEN
                )
            else:
                prog_text = Text("Pending", font_size=FONT_CAPTION, color=NIM_GRAY)
            prog_text.next_to(bg, RIGHT, buff=SPACING_SM)

            row = VGroup(stage_label, bg, fill, prog_text)
            ladder.add(row)

        ladder.arrange(DOWN, buff=SPACING_MD, aligned_edge=LEFT)
        ladder.next_to(ladder_title, DOWN, buff=SPACING_MD)

        self.play(Write(ladder_title))
        self.play(LaggedStart(*[FadeIn(l) for l in ladder], lag_ratio=0.15))

        self.wait(1)
        self.play(FadeOut(Group(*self.mobjects)))

    def play_step_trading(self):
        """Step 5: Trading Terminal."""
        self.play_step_intro("5", "Trading Terminal", "Invest the surplus", NIM_PURPLE)

        # Key message
        key_msg = Text(
            "Only available after your safety net is built",
            font_size=FONT_SMALL,
            color=NIM_ORANGE,
            slant=ITALIC,
        )
        key_msg.to_edge(UP, buff=1.8)
        self.play(Write(key_msg))

        # Available to invest
        available_box = RoundedRectangle(
            width=4.5,
            height=1.8,
            corner_radius=0.2,
            fill_color=NIM_GREEN,
            fill_opacity=0.15,
            stroke_color=NIM_GREEN,
            stroke_width=2,
        )
        available_box.shift(LEFT * 2.5 + UP * 0.2)

        available_content = VGroup(
            Text("Available to invest", font_size=FONT_SMALL, color=NIM_GRAY),
            Text("£125 / week", font_size=FONT_HEADING, color=NIM_GREEN, weight=BOLD),
        ).arrange(DOWN, buff=SPACING_XS)
        available_content.move_to(available_box)

        available = VGroup(available_box, available_content)
        self.play(FadeIn(available, scale=0.9))

        # Portfolio split
        split_title = Text(
            "Portfolio Split", font_size=FONT_BODY, color=WHITE, weight=BOLD
        )
        split_title.shift(RIGHT * 2.5 + UP * 1)

        # Core portfolio
        core_box = RoundedRectangle(
            width=3.5,
            height=1.4,
            corner_radius=0.15,
            fill_color=NIM_BLUE,
            fill_opacity=0.2,
            stroke_color=NIM_BLUE,
            stroke_width=2,
        )
        core_content = VGroup(
            Text("Core Portfolio", font_size=FONT_SMALL, color=NIM_BLUE, weight=BOLD),
            Text("80% - Long term, low risk", font_size=FONT_CAPTION, color=NIM_GRAY),
        ).arrange(DOWN, buff=SPACING_XS)
        core_content.move_to(core_box)
        core = VGroup(core_box, core_content)

        # Explore portfolio
        explore_box = RoundedRectangle(
            width=3.5,
            height=1.4,
            corner_radius=0.15,
            fill_color=NIM_PURPLE,
            fill_opacity=0.2,
            stroke_color=NIM_PURPLE,
            stroke_width=2,
        )
        explore_content = VGroup(
            Text("Explore", font_size=FONT_SMALL, color=NIM_PURPLE, weight=BOLD),
            Text(
                "20% - Active trades, optional", font_size=FONT_CAPTION, color=NIM_GRAY
            ),
        ).arrange(DOWN, buff=SPACING_XS)
        explore_content.move_to(explore_box)
        explore = VGroup(explore_box, explore_content)

        portfolios = VGroup(core, explore).arrange(DOWN, buff=SPACING_SM)
        portfolios.next_to(split_title, DOWN, buff=SPACING_MD)

        self.play(Write(split_title))
        self.play(
            FadeIn(core, shift=LEFT * 0.2),
            FadeIn(explore, shift=LEFT * 0.2),
            lag_ratio=0.2,
        )

        self.wait(1)
        self.play(FadeOut(Group(*self.mobjects)))

    def play_step_autopilot(self):
        """Step 6: Autopilot."""
        self.play_step_intro("6", "Autopilot", "Minimal attention needed", NIM_GRAY)

        # Three frequency cards
        frequencies = [
            ("Daily", "Silent monitoring", "Fraud alerts only", NIM_RED),
            ("Weekly", "60 second digest", "Approve 2 actions", NIM_BLUE),
            ("Monthly", "Full review", "Rebalance check", NIM_GREEN),
        ]

        cards = VGroup()
        for label, line1, line2, color in frequencies:
            card = RoundedRectangle(
                width=3.2,
                height=2.8,
                corner_radius=0.2,
                fill_color="#2C2C2E",
                fill_opacity=1,
                stroke_color=color,
                stroke_width=2,
            )

            label_text = Text(label, font_size=FONT_BODY, color=color, weight=BOLD)
            label_text.move_to(card.get_top() + DOWN * 0.5)

            line1_text = Text(line1, font_size=FONT_SMALL, color=WHITE)
            line2_text = Text(line2, font_size=FONT_CAPTION, color=NIM_GRAY)

            lines = VGroup(line1_text, line2_text).arrange(DOWN, buff=SPACING_XS)
            lines.move_to(card.get_center() + DOWN * 0.2)

            card_group = VGroup(card, label_text, lines)
            cards.add(card_group)

        cards.arrange(RIGHT, buff=SPACING_MD)
        cards.move_to(ORIGIN)

        self.play(LaggedStart(*[FadeIn(c, scale=0.9) for c in cards], lag_ratio=0.15))

        self.wait(1.5)
        self.play(FadeOut(Group(*self.mobjects)))

    def play_flywheel(self):
        """Show the virtuous cycle."""
        title = Text(
            "The Flywheel Effect", font_size=FONT_HEADING, color=WHITE, weight=BOLD
        )
        title.to_edge(UP, buff=SPACING_LG)

        self.play(Write(title))

        # Circular flywheel
        steps = [
            "Stable Budget",
            "Freed Cash",
            "Auto Savings",
            "Safe Surplus",
            "Investing",
            "Wealth",
        ]

        n = len(steps)
        radius = 2.2

        nodes = []
        for i, step in enumerate(steps):
            angle = PI / 2 - (2 * PI * i / n)
            pos = radius * np.array([np.cos(angle), np.sin(angle), 0])

            circle = Circle(
                radius=0.65,
                fill_color=NIM_BLUE,
                fill_opacity=0.25,
                stroke_color=NIM_BLUE,
                stroke_width=2,
            )
            circle.move_to(pos)

            # Split text if needed for better fit
            text = Text(step, font_size=FONT_CAPTION, color=WHITE)
            text.move_to(circle)

            node = VGroup(circle, text)
            nodes.append(node)

        flywheel = VGroup(*nodes)
        flywheel.move_to(ORIGIN)

        self.play(LaggedStart(*[FadeIn(n, scale=0.5) for n in nodes], lag_ratio=0.1))

        # Animated connecting arrows
        arrows = []
        for i in range(n):
            start = nodes[i][0].get_center()
            end = nodes[(i + 1) % n][0].get_center()

            arrow = CurvedArrow(
                start, end, color=NIM_GREEN, stroke_width=3, angle=TAU / 10
            )
            arrows.append(arrow)

        for arrow in arrows:
            self.play(Create(arrow), run_time=0.25)

        # Gentle rotation to show the cycle
        all_elements = VGroup(flywheel, *arrows)
        self.play(
            Rotate(all_elements, angle=TAU / 8, about_point=ORIGIN),
            run_time=2,
            rate_func=smooth,
        )

        self.wait(1)
        self.play(FadeOut(Group(*self.mobjects)))

    def play_cta(self):
        """Call to action."""
        logo = Text("Nim", font_size=72, weight=BOLD, color=WHITE)
        logo.set_color_by_gradient(NIM_BLUE, NIM_PURPLE)

        tagline = Text("Money, Multiplied", font_size=FONT_SUBHEADING, color=NIM_GREEN)
        tagline.next_to(logo, DOWN, buff=SPACING_MD)

        cta = Text(
            "Start your journey today",
            font_size=FONT_BODY,
            color=NIM_GRAY,
            slant=ITALIC,
        )
        cta.next_to(tagline, DOWN, buff=SPACING_XL)

        group = VGroup(logo, tagline, cta)
        group.move_to(ORIGIN)

        self.play(FadeIn(logo, scale=0.8), run_time=1)
        self.play(FadeIn(tagline, shift=UP * 0.3))
        self.play(FadeIn(cta, shift=UP * 0.2))

        self.wait(2)
        self.play(FadeOut(group))

    def play_step_intro(self, number, title, subtitle, color):
        """Reusable step introduction header."""
        # Number badge
        circle = Circle(radius=0.35, fill_color=color, fill_opacity=1, stroke_width=0)
        num = Text(number, font_size=FONT_SUBHEADING, color=WHITE, weight=BOLD)
        num.move_to(circle)
        num_group = VGroup(circle, num)

        # Title
        title_text = Text(title, font_size=FONT_HEADING, color=WHITE, weight=BOLD)
        title_text.next_to(num_group, RIGHT, buff=SPACING_SM)

        # Subtitle
        sub_text = Text(subtitle, font_size=FONT_BODY, color=NIM_GRAY)
        sub_text.next_to(title_text, DOWN, buff=SPACING_XS, aligned_edge=LEFT)

        header = VGroup(num_group, title_text, sub_text)
        header.to_edge(UP, buff=SPACING_LG).to_edge(LEFT, buff=1.2)

        self.play(
            FadeIn(num_group, scale=0.5),
            FadeIn(title_text, shift=LEFT * 0.3),
        )
        self.play(FadeIn(sub_text, shift=UP * 0.1))
        self.wait(0.2)


class NimFlowDiagram(Scene):
    """A simpler flow diagram animation."""

    def construct(self):
        self.camera.background_color = NIM_DARK

        title = Text(
            "The Nim Journey", font_size=FONT_HEADING, color=WHITE, weight=BOLD
        )
        title.to_edge(UP, buff=SPACING_LG)
        self.play(Write(title))

        # Flow boxes
        steps = [
            ("Chat", NIM_BLUE),
            ("Budget", NIM_TEAL),
            ("Subs", NIM_ORANGE),
            ("Save", NIM_GREEN),
            ("Invest", NIM_PURPLE),
        ]

        boxes = []
        for name, color in steps:
            box = RoundedRectangle(
                width=1.8,
                height=1.2,
                corner_radius=0.15,
                fill_color=color,
                fill_opacity=0.25,
                stroke_color=color,
                stroke_width=3,
            )

            name_text = Text(name, font_size=FONT_SMALL, color=WHITE, weight=BOLD)
            name_text.move_to(box)

            group = VGroup(box, name_text)
            boxes.append(group)

        flow = VGroup(*boxes).arrange(RIGHT, buff=SPACING_SM)
        flow.move_to(ORIGIN)

        self.play(LaggedStart(*[FadeIn(b, scale=0.8) for b in boxes], lag_ratio=0.12))

        # Arrows
        arrows = []
        for i in range(len(boxes) - 1):
            arrow = Arrow(
                boxes[i].get_right(),
                boxes[i + 1].get_left(),
                buff=0.1,
                color=WHITE,
                stroke_width=2,
            )
            arrows.append(arrow)

        self.play(LaggedStart(*[GrowArrow(a) for a in arrows], lag_ratio=0.08))

        # Autopilot loop
        loop_arrow = CurvedArrow(
            boxes[-1].get_bottom() + DOWN * 0.15,
            boxes[0].get_bottom() + DOWN * 0.15,
            color=NIM_GRAY,
            stroke_width=2,
            angle=-TAU / 5,
        )

        loop_label = Text("Autopilot", font_size=FONT_CAPTION, color=NIM_GRAY)
        loop_label.next_to(loop_arrow, DOWN, buff=SPACING_XS)

        self.play(Create(loop_arrow), FadeIn(loop_label))

        self.wait(2)


if __name__ == "__main__":
    print("Run with: manim -pql nim_onboarding.py NimOnboarding")
