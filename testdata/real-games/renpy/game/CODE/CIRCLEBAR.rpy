## КРУГЛЫЙ БАР

# круглый бар отображает переменную v
# range - максимальное значение переменной
# size - размеры круглого бара
# t = 0 - статичный bar
# t = 1.5 и start=0 - bar за полторы секунды заполнится с 0 до range
# t = 1 и start=1 - bar за секунду опустеет

# пример:
# add CircleBar(v="cb_value", range=1, size=(150, 150), t=0, start=0)

# части бара
image cb_bg = Image("images/circle_bar/bg.png")
image cb_bgzone = Image("images/circle_bar/bgzone.png")
image cb_bgzone_half = Image("images/circle_bar/bgzone_half.png")
image cb_dot = Image("images/circle_bar/dot.png")
image cb_full = Image("images/circle_bar/full.png")
image cb_half = Image("images/circle_bar/half.png")
image cb_mask = Image("images/circle_bar/mask.png")

# переменная по умолчанию
default cb_value = 0

init python:
    # аналог Composite, но без указания позиции, все кадры в центре контейнера
    def CComp(size, *args, **kwarg):
        rv = renpy.display.layout.Fixed(xysize=size, **kwarg)
        kwarg.setdefault('anchor', (.5, .5))

        for widget in args:
            x, y = kwarg.get("anchor")
            rv.add(renpy.display.layout.Position(widget, xpos=x, xanchor=x, ypos=y, yanchor=y))

        return rv

    def rotsize(image, size=(150, 150), angle=.0001):
        return Transform(image, xysize=size, rotate=angle)

    # круглый бар отображает переменную v
    # t = 0 - статичный bar
    # t = 1.5 и start=0 - bar за полторы секунды заполнится с 0 до range
    # t = 1 и start=1 - bar за секунду опустеет
    def circle_bar_f(st, at, v="cb_value", range=1, size=(150, 150), t=1, start=0, a1=270, a2=360):
        value = eval(v)
        value = max(0, min(range, value))

        if t:
            value = range * min(st / t, 1)
            globals()[v] = value

        angle = a2 - a1

        w, h = size
        args = [ size, rotsize("cb_bg", size) ]

        if not t is None:
            if 0 < angle <= 180:
                args.append( Transform(AlphaBlend(rotsize("cb_mask", size, angle), rotsize("cb_bgzone_half", size), Solid("#0000"), alpha=True), rotate=a1))

            elif angle > 180:
                args.append( Transform(AlphaBlend(rotsize("cb_mask", size, angle), rotsize("cb_bgzone", (w + 1, h)), Solid("#0000"), alpha=True), rotate=a1) )
                args.append( Transform(rotsize("cb_bgzone_half", size), rotate=a1) )

        angle = abs(start - min(1, value / range)) * 360

        if 0 < angle <= 180:
            args.append( AlphaBlend(rotsize("cb_mask", size, angle), rotsize("cb_half", size), Solid("#0000"), alpha=True) )

        elif angle > 180:
            args.append( AlphaBlend(rotsize("cb_mask", size, angle), rotsize("cb_full", (w + 1, h)), Solid("#0000"), alpha=True) )
            args.append( rotsize("cb_half", size) )

        args.append( rotsize("cb_dot", size, angle) )

        return Flatten(CComp(*args)), 1. / config.framerate

    # круглый бар
    def CircleBar(v="cb_value", range=1, size=(150, 150), t=1, start=0, a1=270, a2=360):
        return DynamicDisplayable(circle_bar_f, v, range, size, t, start, a1, a2)

# ТЕСТЫ
screen test_circle_slider():
    timer .0001 action SetVariable("cb_value", 0)

    vbox align (.5, .5):
        fixed fit_first True:
            add CircleBar()

        null height 20
        bar value VariableValue("cb_value", 360) xsize 150

screen test_circle():
    add CircleBar(t=4.75) align (.5, .5)

label test_circle:
    show screen test_circle
    pause

    $ xxx = cb_value

    hide screen test_circle

    "[xxx]"

    return