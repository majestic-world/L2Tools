using System.Text;

const string alphabet = "abcdefghijklmnopqrstuvwxyz";

Console.Title = "ProGuard Keygen v1.0";
Console.WriteLine("=== Pro Guard Dictionary Generator ===");
Console.WriteLine();

var count = ReadInt("How many combinations to generate? ");
var minLength = ReadInt("Minimum word length? ");
var maxLength = ReadMinMax("Maximum word length? ", minLength);

Console.WriteLine();
Console.WriteLine($"Generating {count} unique words (length {minLength}-{maxLength})...");

var generated = new HashSet<string>(count);
var random = new Random();
var sb = new StringBuilder(maxLength);

while (generated.Count < count)
{
    var length = random.Next(minLength, maxLength + 1);
    sb.Clear();
    for (var i = 0; i < length; i++)
    {
        var c = alphabet[random.Next(alphabet.Length)];
        sb.Append(i > 0 && random.NextDouble() < 0.40 ? char.ToUpper(c) : c);
    }

    generated.Add(sb.ToString());
}

var header = new[]
{
    $"# ╔══════════════════════════════════╗",
    $"# ║   Pro Guard Keygen  ·  By Mk     ║",
    $"# ╚══════════════════════════════════╝",
    $"# Words: {count}  |  Length: {minLength}-{maxLength}",
    $"",
};

var words = generated
    .Select((word, i) => (word, i))
    .GroupBy(x => x.i / 10)
    .Select(g => string.Join(" ", g.Select(x => x.word)));

var outputPath = Path.Combine(AppContext.BaseDirectory, "output.txt");
File.WriteAllLines(outputPath, header.Concat(words));

Console.WriteLine();
Console.WriteLine($"Done! {count} words saved to:");
Console.WriteLine($"{outputPath}");
Console.WriteLine();
Console.Write("Press any key to exit...");
Console.ReadKey(intercept: true);

static int ReadInt(string prompt)
{
    while (true)
    {
        Console.Write(prompt);
        string? input = Console.ReadLine();
        if (int.TryParse(input, out int value) && value > 0)
            return value;
        Console.WriteLine("Invalid value. Enter a positive integer.");
    }
}

static int ReadMinMax(string prompt, int min)
{
    while (true)
    {
        Console.Write(prompt);
        var input = Console.ReadLine();
        if (int.TryParse(input, out int value) && value >= min)
            return value;
        Console.WriteLine($"Invalid value. Must be >= {min}.");
    }
}
